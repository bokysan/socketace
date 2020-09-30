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

PACKAGE="github.com/bokysan/socketace/v2/internal/version"
GOOS="$(echo "$TARGETPLATFORM" | cut -f1 -d/)"
GOARM="$(echo "$TARGETPLATFORM" | cut -f3 -d/)"
GOARCH="$(echo "$TARGETPLATFORM" | cut -f2 -d/)"
GIT_COMMIT="-X $PACKAGE.GitCommit=$(git rev-parse HEAD)"
GIT_BRANCH="-X $PACKAGE.GitBranch=$(git symbolic-ref --short HEAD)"
GIT_TAG="-X $PACKAGE.GitBranch=$(git tag --points-at HEAD)"
GIT_SUMMARY="-X $PACKAGE.GitSummary=$(git describe --tags --dirty --always)"

case "$GIT_SUMMARY" in
  *dirty*)
    GIT_STATE="-X $PACKAGE.GitState=dirty"
    ;;
  *)
    GIT_STATE="-X $PACKAGE.GitState=clean"
esac


BUILD_DATE="-X $PACKAGE.BuildDate=$(date -u +\"%Y-%m-%dT%H:%M:%SZ\")"
GO_VERSION="-X $PACKAGE.GoVersion=$(go version)"


export PACKAGE
export GOOS
export GOARM
export GOARCH
export GIT_COMMIT
export GIT_BRANCH
export GIT_TAG
export GIT_SUMMARY
export GIT_STATE
export BUILD_DATE
export GO_VERSION

mkdir -p target

if [ -z "$OUTPUT" ]; then
  OUTPUT="target/socketace-$GOOS-$GOARCH"
  if [ -n "$GOARM" ]; then
    OUTPUT="$OUTPUT-$GOARM"
  fi
  if [ "$GOOS" = "windows" ]; then
    OUTPUT="$OUTPUT.exe"
  fi
fi

echo "Building: $GOOS/$GOARCH: $OUTPUT"
go build \
    -o "$OUTPUT" \
    -ldflags "-extldflags '-static'" \
    cmd/socketace/main.go