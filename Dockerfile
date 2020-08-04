# build method: just run `docker build --rm --build-arg -t genshen/wssocks .`

# build frontend code
FROM node:12-alpine AS web-builder

COPY status-web web/

RUN cd web \
    && yarn install \
    && yarn build

## build go binary
FROM golang:1.14.6-alpine AS builder

ARG PACKAGE=github.com/genshen/wssocks

RUN apk add --no-cache git \
    && go get -u github.com/rakyll/statik

COPY ./  /go/src/${PACKAGE}/
COPY --from=web-builder web/build /go/src/github.com/genshen/wssocks/web-build/

RUN cd ./src/${PACKAGE} \
    && cd server \
    && statik -src=../web-build \
    && cd ../ \
    && go build \
    && go install

## copy binary
FROM alpine:latest

ARG HOME="/home/wssocks"

RUN adduser -D wssocks -h ${HOME}

COPY --from=builder --chown=wssocks /go/bin/wssocks ${HOME}/wssocks

WORKDIR ${HOME}
USER wssocks

ENTRYPOINT ["./wssocks"]
CMD ["--help"]
