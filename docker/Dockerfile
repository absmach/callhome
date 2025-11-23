FROM golang:1.25.4-alpine3.22 AS builder
ARG SVC
ARG GOARCH
ARG GOARM
ARG VERSION
ARG COMMIT
ARG TIME

WORKDIR /go/src/github.com/absmach/callhome
COPY . .
RUN go mod tidy
RUN apk update \
    && apk add make npm\
    && make $SVC \
    && mv build/$SVC /exe

# Build minified assets
RUN npm install && npm run build

FROM scratch
# Certificates are needed so that mailing util can work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
COPY --from=builder /go/src/github.com/absmach/callhome/web /web
ENTRYPOINT ["/exe"]
