#!/usr/bin/env bash

build_args() {
	local build_env="$1"
	local prefix="$2"
	local name
	local val

	if [ -f "$build_env" ]; then
		cat "$build_env" \
			| sed -e '/^[ \t]*#/d' \
			| sed -e '/^[ \t]*$/d' \
			| sed -E "s/'/\\\\\'/g" \
			| sed -E 's/^([A-Za-z_][A-Za-z0-9_-]+)=(.+)$/\1='"'"'\2'"'"'/' \
			| sed -E "s/^/$prefix/g" \
			| sed -e ':a' -e 'N' -e '$!ba' -e "s/\n/ /g" \
			| cat
	fi
}

create_buildx_environment() {
	if ! docker buildx inspect multiarch > /dev/null; then
			docker buildx create --name multiarch
	fi
	docker buildx use multiarch
}

docker_login() {
	if [[ -n "$DOCKER_USERNAME" ]] && [[ -n "$DOCKER_PASSWORD" ]]; then
			echo "Logging into docker registry $DOCKER_REGISTRY_URL...."
			echo "$DOCKER_PASSWORD" | docker login --username $DOCKER_USERNAME --password-stdin $DOCKER_REGISTRY_URL
	fi
}

setup_default_platforms() {
	if [[ -z "$PLATFORMS" ]]; then
		PLATFORMS="linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64,linux/ppc64le"
	fi
}