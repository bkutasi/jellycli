# Base image for building purposes
FROM docker.io/golang:1.15.5-alpine3.12 as builder

RUN apk --no-cache add alsa-lib-dev git alpine-sdk

WORKDIR /jellycli

ARG JELLYCLI_BRANCH=master

# use caching layers as needed

# Copy local source code instead of cloning
COPY . .

RUN go mod download

RUN go build . && ./jellycli --help


# Alpine runtime
FROM docker.io/alpine:3.12

RUN apk --no-cache add alsa-lib-dev dbus-x11 alsa-utils
COPY --from=builder /jellycli/jellycli /usr/local/bin

RUN mkdir /root/.config/
ENTRYPOINT [ "jellycli" ]
