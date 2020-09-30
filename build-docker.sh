#!/usr/bin/env bash
set -e

declare SCRIPT_PATH
SCRIPT_PATH="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

# shellcheck source=./functions.sh
source "${SCRIPT_PATH}/functions.sh"

# Do a multistage build
export DOCKER_BUILDKIT=1
export DOCKER_CLI_EXPERIMENTAL=enabled

if [[ "$*" == *--push* ]]; then
	docker_login
fi

setup_default_platforms
docker buildx build --platform $PLATFORMS . $*