# Copyright (c) Abstract Machines

PROGRAM = callhome
MG_DOCKER_IMAGE_NAME_PREFIX ?= supermq
SOURCES = $(wildcard *.go) cmd/main.go
CGO_ENABLED ?= 0
GOARCH ?= amd64
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo "0.13.0")
COMMIT ?= $(shell git rev-parse HEAD)
TIME ?= $(shell date +%F_%T)
DOMAIN ?= deployments.absmach.eu

all: $(PROGRAM)

.PHONY: all clean $(PROGRAM) latest build-assets

define make_docker
	docker build \
		--no-cache \
		--build-arg SVC=$(PROGRAM) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg GOARM=$(GOARM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg TIME=$(TIME) \
		--tag=$(MG_DOCKER_IMAGE_NAME_PREFIX)/$(PROGRAM) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	docker build \
		--no-cache \
		--build-arg SVC=$(PROGRAM) \
		--tag=$(MG_DOCKER_IMAGE_NAME_PREFIX)/$(PROGRAM) \
		-f docker/Dockerfile.dev .
endef

define make_dev_cert
	mkdir -p ./docker/certbot/conf/live/$(DOMAIN)
	[ -f ./docker/certbot/conf/options-ssl-nginx.conf ] || curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > ./docker/certbot/conf/options-ssl-nginx.conf
	[ -f ./docker/certbot/conf/ssl-dhparams.pem ] || curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > ./docker/certbot/conf/ssl-dhparams.pem
	openssl req -x509 -out ./docker/certbot/conf/live/$(DOMAIN)/fullchain.pem \
	-keyout ./docker/certbot/conf/live/$(DOMAIN)/privkey.pem \
	-newkey rsa:2048 -nodes -sha256 \
	-subj '/CN=localhost'
endef

$(PROGRAM): $(SOURCES)
	npm run
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -ldflags "-s -w \
	-X 'github.com/absmach/callhome.BuildTime=$(TIME)' \
	-X 'github.com/absmach/callhome.Version=$(VERSION)' \
	-X 'github.com/absmach/callhome.Commit=$(COMMIT)'" \
	-o ./build/$(PROGRAM) cmd/main.go

clean:
	rm -rf build

cleandocker:
	docker compose -f ./docker/docker-compose.yml down --rmi all -v --remove-orphans

docker-image:
	$(call make_docker)

docker-dev:
	$(call make_docker_dev)

dev-cert:
	$(call make_dev_cert)

run:
	docker compose -f ./docker/docker-compose.yml up

test:
	go test -v --race -count=1 -failfast -covermode=atomic -coverprofile cover.out ./...

build-assets:
	./docker/build-assets.sh

latest: docker-image
	docker push supermq/callhome:latest
