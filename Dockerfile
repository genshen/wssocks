# build method: just run `docker build --rm --build-arg -t genshen/wssocks .`

FROM golang:1.12.4-alpine AS builder

# set to 'on' if using go module
ENV GO111MODULE=on
ARG PACKAGE=github.com/genshen/wssocks

RUN apk add --no-cache git

COPY ./  /go/src/${PACKAGE}/

RUN cd ./src/${PACKAGE} \
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
