# Base image for building purposes
FROM docker.io/golang:1.24.2-alpine3.21 AS builder

RUN apk --no-cache add alsa-lib-dev alpine-sdk

WORKDIR /jellycli

ARG JELLYCLI_BRANCH=master

# use caching layers as needed

# Copy go module files and download dependencies first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application, stripping debug symbols
RUN go build -ldflags="-s -w" -o jellycli .


# Alpine runtime
FROM docker.io/alpine:3.21

RUN apk --no-cache add alsa-lib
COPY --from=builder /jellycli/jellycli /usr/local/bin/jellycli

RUN mkdir /root/.config/
ENTRYPOINT [ "jellycli" ]
