FROM golang:1.19-alpine AS builder
ARG SVC
ARG GOARCH
ARG GOARM
ARG VERSION
ARG COMMIT
ARG TIME

WORKDIR /go/src/github.com/mainflux/et
COPY . .
RUN apk update \
    && apk add make\
    && make $SVC \
    && mv build/et-$SVC /exe

FROM scratch
# Certificates are needed so that mailing util can work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]