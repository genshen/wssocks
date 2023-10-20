# build method: just run `docker build --rm -t genshen/wssocks .`

# build frontend code
FROM node:18.18.2-alpine3.18 AS web-builder

COPY status-web web/

RUN cd web \
    && yarn install \
    && yarn build

## build go binary
FROM golang:1.21.3-alpine3.18 AS builder

ARG PACKAGE=github.com/genshen/wssocks
ARG BUILD_FLAG="-X 'github.com/genshen/wssocks/version.buildHash=`git rev-parse HEAD`' \
 -X 'github.com/genshen/wssocks/version.buildTime=`date`' \
 -X 'github.com/genshen/wssocks/version.buildGoVersion=`go version | cut -f 3,4 -d\" \"`'"

RUN apk add --no-cache git \
    && go install github.com/rakyll/statik@v0.1.7

COPY ./  /go/src/${PACKAGE}/
COPY --from=web-builder web/build /go/src/github.com/genshen/wssocks/web-build/

RUN cd ./src/${PACKAGE} \
    && cd cmd/server \
    && statik -src=../../web-build \
    && cd ../../ \
    && go build -ldflags "${BUILD_FLAG}" -o wssocks ${PACKAGE} \
    && go install

## copy binary
FROM alpine:3.18.0

ARG HOME="/home/wssocks"

RUN adduser -D wssocks -h ${HOME}

COPY --from=builder --chown=wssocks /go/bin/wssocks ${HOME}/wssocks

WORKDIR ${HOME}
USER wssocks

ENTRYPOINT ["./wssocks"]
CMD ["--help"]
