FROM golang:1.17 as build-env

ADD . /go/src/github.com/miseyu/go-sftp
WORKDIR /go/src/github.com/miseyu/go-sftp

ARG CGO_ENABLED=0

RUN go mod vendor
RUN go build -ldflags "-s -w" -o /go/bin/app

FROM alpine:3.16
COPY --from=build-env /go/bin/app /

ENTRYPOINT ["/app"]
