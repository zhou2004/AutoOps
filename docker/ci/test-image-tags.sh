#!/usr/bin/env bash
set -euo pipefail
output=$(mktemp)
trap 'rm -f "$output"' EXIT
export IMAGE_REPOSITORY=feelzhou/autoops COMPONENT=api
export GITHUB_SHA=0123456789abcdef0123456789abcdef01234567 GITHUB_OUTPUT="$output"
export GITHUB_REF_TYPE=branch CUSTOM_TAG=''
bash docker/ci/image-tags.sh
grep -Fx 'feelzhou/autoops:api-sha-0123456789ab' "$output"
export COMPONENT=web CUSTOM_TAG=v1.2.3
bash docker/ci/image-tags.sh
grep -Fx 'feelzhou/autoops:web-v1.2.3' "$output"
export GITHUB_REF_TYPE=tag GITHUB_REF_NAME=v2.0.0 CUSTOM_TAG=ignored
bash docker/ci/image-tags.sh
grep -Fx 'feelzhou/autoops:web-v2.0.0' "$output"
export GITHUB_REF_TYPE=branch CUSTOM_TAG='bad tag'
if bash docker/ci/image-tags.sh >/dev/null 2>&1; then echo 'Invalid tag was accepted' >&2; exit 1; fi
export CUSTOM_TAG='$(false)'
if bash docker/ci/image-tags.sh >/dev/null 2>&1; then echo 'Shell input was accepted' >&2; exit 1; fi
export CUSTOM_TAG='' COMPONENT=invalid
if bash docker/ci/image-tags.sh >/dev/null 2>&1; then echo 'Invalid component was accepted' >&2; exit 1; fi
echo 'Image tag checks passed'
