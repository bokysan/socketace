#!/usr/bin/env bash

#
# Extract the data on which docker image tags should be built from the list of git tags
#

set -o errexit
set -o pipefail
set -o nounset

declare TAG
declare ALL_TAGS
declare FOLLOWING_TAGS

# version_weight converts the semver versions into a list which can be compared using `sort -V`
version_weight () {
  echo -e "$1" | tr ' ' "\n"  | sed -e 's:\+.*$::' | sed -e 's:^v::' | \
  sed -Ee 's:^[0-9]+(\.[0-9]+)+$:&-stable:' | \
  sed -Ee 's:([^A-Za-z])dev\.?([^A-Za-z]|$):\1.10.\2:g' | \
  sed -Ee 's:([^A-Za-z])(alpha|a)\.?([^A-Za-z]|$):\1.20.\3:g' | \
  sed -Ee 's:([^A-Za-z])(beta|b)\.?([^A-Za-z]|$):\1.30.\3:g' | \
  sed -Ee 's:([^A-Za-z])(rc|RC)\.?([^A-Za-z]|$)?:\1.40.\3:g' | \
  sed -Ee 's:([^A-Za-z])stable\.?([^A-Za-z]|$):\1.50.\2:g' | \
  sed -Ee 's:([^A-Za-z])pl\.?([^A-Za-z]|$):\1.60.\2:g' | \
  sed -Ee 's:([^A-Za-z])(patch|p)\.?([^A-Za-z]|$):\1.70.\3:g' | \
  sed -E 's:\.{2,}:.:' | \
  sed -E 's:\.$::' | \
  sed -E 's:-\.:.:' | \
  sed -E 's:^:v:'
}


# semversort will take a list of semantic versions and sort them properly. It takes a list of versions either
# as list of arguments or as an input stream. One tag per line.
# shellcheck disable=SC2120
semversort() {
  local -a tags_orig
  local -a tags_weight
  local -a versions_list
  local -a keys

  if [[ $# -gt 0 ]]; then
    versions_list=$*
  else
    # catch pipeline output
    versions_list=$(cat)
  fi

  # Map the tags into an array
  mapfile -t tags_orig < <(echo "${versions_list}")

  # Create a restructured list of versions which can be sorted
  mapfile -t tags_weight < <(version_weight "${tags_orig[*]}")

  # Calculate the key (index) values on the original array
  keys=$(for ix in ${!tags_weight[*]}; do
      printf "%s+%s\n" "${tags_weight[${ix}]}" "${ix}"
  done | sort -V | cut -d+ -f2)

  # Return data from the original array, sorted in ascending order
  for ix in ${keys}; do
      printf "%s\n" "${tags_orig[${ix}]}"
  done
}

# Read the GitHub reference (we expect it to be in format "ref/tags/<tag-name>"
TAG="${GITHUB_REF:10}"

if [[ "$TAG" == v* ]]; then
  # Get a list of all tags on the project, sort it by tag semantic version
  ALL_TAGS="$(git tag -l | grep -E "^v*" | semversort)"

  # Get a list of tags following our tag. List might be empty if this is the latest tag
  FOLLOWING_TAGS="$(echo "$ALL_TAGS" | sed -e "1,/^${TAG}/ d")"

  # Initialize the tag array
  TAGS=""
  if [[ -z "$FOLLOWING_TAGS" ]]; then
    TAGS="-t boky/socketace:latest"
  fi

  # Gradually split the tag
  # TODO: Also take into account that the tag may be followed by optional ["-" <pre-release>]["+" <build>]
  while [[ -n "$TAG" ]]; do
    if ! echo "${FOLLOWING_TAGS}" | grep -E -q "$TAG"; then
      TAGS="$TAGS -t boky/socketace:$TAG"
    fi
    NEW_TAG="$(echo "$TAG" | rev | cut -d. -f2- | rev)"
    if [[ "$NEW_TAG" == "$TAG" ]]; then
      break
    else
      TAG="$NEW_TAG"
    fi
  done
  echo "::set-env name=TAGS::$TAGS"
else
  echo "::set-env name=DO_BUILD_TAG::"
fi

echo "::set-env name=DO_BUILD_TAG::1"

