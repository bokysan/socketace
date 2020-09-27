# ================ BUILD EXECUTABLE MODULE ================
FROM --platform=$BUILDPLATFORM golang:1.14-alpine AS build
ARG BUILDPLATFORM
LABEL maintainer="Bojan Cekrlic <https://github.com/bokysan/>"

RUN apk add --no-cache bash git
RUN mkdir -p /usr/local/go/src/github.com/bokysan/socketace
WORKDIR /usr/local/go/src/github.com/bokysan/socketace

COPY go.mod build.sh ./
RUN go mod download

COPY .git ./.git
COPY cmd ./cmd
COPY internal ./internal

ARG TARGETPLATFORM
RUN echo "Building on $BUILDPLATFORM for $TARGETPLATFORM"
RUN ./build.sh

# ================ COMPRESS EXECUTABLE MODULE ================
FROM --platform=$TARGETPLATFORM alpine AS upx
ARG TARGETPLATFORM

RUN apk add --no-cache upx

COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
RUN upx -9 /bin/socketace && upx -t /bin/socketace
RUN /bin/socketace version

# ================ BUILD FINAL IMAGE ================
FROM --platform=$TARGETPLATFORM scratch
COPY --from=upx /bin/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]