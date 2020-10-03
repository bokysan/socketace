# ================ BUILD EXECUTABLE MODULE ================
FROM --platform=$BUILDPLATFORM golang:1.15-alpine AS build
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
    sed -i -e "s/^    goos:.*# Dynamic\$/    goos: [ '$GOOS' ]/" .goreleaser.yml && \
    sed -i -e "s/^    goarch:.*# Dynamic\$/    goarch: [ '$GOARCH' ]/" .goreleaser.yml && \
    sed -i -e "s/^    goarm:.*# Dynamic\$/    goarm: [ '$GOARM' ]/" .goreleaser.yml && \
    sed -i -e "s/^    gomips:.*# Dynamic\$/    gomips: [ 'softfloat' ]/" .goreleaser.yml && \
    ./bin/goreleaser build --rm-dist --skip-validate $GORELEASER_EXTRA_ARGS && \
    export DIR="default_${GOOS}_${GOARCH}" && \
    if [ -n "${GOARM}" ]; then export DIR="${DIR}_${GOARM}"; fi && \
    case "${GOARCH}" in mips*) export DIR="${DIR}_softfloat"; ;; esac && \
    cp dist/${DIR}/socketace ./socketace

# ================ linux/386 ================
FROM --platform=linux/386 alpine AS upx
RUN apk add --no-cache upx
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
RUN upx -9 /bin/socketace
RUN /bin/socketace version

FROM --platform=linux/386 scratch
COPY --from=upx /bin/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/amd64 ================
FROM --platform=linux/amd64 alpine AS upx
RUN apk add --no-cache upx
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
RUN upx -9 /bin/socketace
RUN /bin/socketace version

FROM --platform=linux/amd64 scratch
COPY --from=upx /bin/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/arm/v5 ================
FROM --platform=linux/arm/v5 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/arm/v6 ================
FROM --platform=linux/arm/v6 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/arm/v7 ================
FROM --platform=linux/arm/v7 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/ppc64 ================
FROM --platform=linux/ppc64 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/ppc64le ================
FROM --platform=linux/ppc64le scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/ppc64le ================
FROM --platform=linux/ppc64le scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mips ================
FROM --platform=linux/mips scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mipsle ================
FROM --platform=linux/mipsle scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mips32 ================
FROM --platform=linux/mips32 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mips32le ================
FROM --platform=linux/mips32le scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mips64 ================
FROM --platform=linux/mips64 scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/mips64le ================
FROM --platform=linux/mips64le scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]

# ================ linux/s390x ================
FROM --platform=linux/s390x scratch
COPY --from=build /usr/local/go/src/github.com/bokysan/socketace/socketace /bin/socketace
ENTRYPOINT [ "/bin/socketace" ]
