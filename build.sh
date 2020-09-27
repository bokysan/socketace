#!/bin/sh

export GO111MODULE=on
export CGO_ENABLED=0

#	GitCommit  string // long commit hash of source tree, e.g. "0b5ed7a"
#	GitBranch  string // current branch name the code is built off, e.g. "master"
#	GitTag     string // current tag name the code is built off, e.g. "v1.5.0"
#	GitSummary string // output of "git describe --tags --dirty --always", e.g. "4cb95ca-dirty"
#	GitState   string // whether there are uncommitted changes, e.g. "clean" or "dirty"
#	BuildDate  string // RFC3339 formatted UTC date, e.g. "2016-08-04T18:07:54Z"
#	Version    string // contents of ./VERSION file, if exists
#	GoVersion  string // the version of go, e.g. "go version go1.10.3 darwin/amd64"
#	ProtoVersion string = "v1.0.0"

export PACKAGE="github.com/bokysan/socketace/v2/internal/version" && \
export GOOS="$(echo "$TARGETPLATFORM" | cut -f1 -d/)"
export GOARM="$(echo "$TARGETPLATFORM" | cut -f3 -d/ | cut -c2-)"
export GOARCH="$(echo "$TARGETPLATFORM" | cut -f2 -d/)"
export GIT_COMMIT="-X $PACKAGE.GitCommit=$(git rev-parse HEAD)"
export GIT_BRANCH="-X $PACKAGE.GitBranch=$(git symbolic-ref --short HEAD)"
export GIT_TAG="-X $PACKAGE.GitBranch=$(git tag --points-at HEAD)"
export GIT_SUMMARY="-X $PACKAGE.GitSummary=$(git describe --tags --dirty --always)"
export GIT_STATE="-X $PACKAGE.GitSummary=$(git describe --tags --dirty --always)"
export BUILD_DATE="-X $PACKAGE.BuildDate=$(date -u +\"%Y-%m-%dT%H:%M:%SZ\")"
export GO_VERSION="-X $PACKAGE.GoVersion=$(go version)"

echo "Building: $GOOS/$GOARCH"
go build \
    -o socketace \
    -ldflags "-extldflags '-static'" \
    cmd/socketace/main.go