# ================ BUILD EXECUTABLE MODULE ================
FROM --platform=$BUILDPLATFORM golang:1.14-alpine AS build
ARG BUILDPLATFORM
LABEL maintainer="Bojan Cekrlic <https://github.com/bokysan/>"

RUN apk add --no-cache bash git sed curl
RUN mkdir -p /usr/local/go/src/github.com/bokysan/socketace
WORKDIR /usr/local/go/src/github.com/bokysan/socketace
RUN curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

COPY .goreleaser.yml go.mod ./
RUN go mod download

COPY .git ./.git
COPY cmd ./cmd
COPY internal ./internal

ARG TARGETPLATFORM
ARG GORELEASER_EXTRA_ARGS
RUN true && \
    echo "Building on $BUILDPLATFORM for $TARGETPLATFORM" && \
    export GOOS="$(echo "$TARGETPLATFORM" | cut -f1 -d/)" && \
    export GOARCH="$(echo "$TARGETPLATFORM" | cut -f2 -d/)" && \
    export GOARM="$(echo "$TARGETPLATFORM" | cut -f3 -d/ | sed -e 's/^v//')" && \
    export GOVERSION="$(go version)" && \
    export GIT_BRANCH="$(git symbolic-ref --short HEAD 2>/dev/null || echo '')" && \
    sed -i -e "s/^    goos:.*\$/    goos: [ '$GOOS' ]/" .goreleaser.yml && \
    sed -i -e "s/^    goarch:.*\$/    goarch: [ '$GOARCH' ]/" .goreleaser.yml && \
    sed -i -e "s/^    goarm:.*\$/    goarm: [ '$GOARM' ]/" .goreleaser.yml && \
    ./bin/goreleaser build --rm-dist --skip-validate $GORELEASER_EXTRA_ARGS && \
    export DIR="default_${GOOS}_${GOARCH}" && \
    if [ -n "${GOARM}" ]; then export DIR="${DIR}_${GOARM}"; fi && \
    cp dist/${DIR}/socketace ./socketace

# ================ COMPRESS EXECUTABLE MODULE ================
FROM --platform=$TARGETPLATFORM alpine AS upx
ARG TARGETPLATFORM

RUN case "$TARGETPLATFORM" in "linux/ppc64le") ;; "linux/arm/v6") ;; "linux/arm/v7") ;; "linux/arm64") ;; *) apk add --no-cache upx ;; esac

COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
RUN case "$TARGETPLATFORM" in "linux/ppc64le") ;; "linux/arm/v6") ;; "linux/arm/v7") ;; "linux/arm64") ;; *) upx -9 /bin/socketace && upx -t /bin/socketace ;; esac
RUN /bin/socketace version

# ================ BUILD FINAL IMAGE ================
FROM --platform=$TARGETPLATFORM scratch
COPY --from=upx /bin/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]