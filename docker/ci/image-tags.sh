#!/usr/bin/env bash
# Generate component-scoped tags without interpolating user input into shell code.
set -euo pipefail
: "${IMAGE_REPOSITORY:?}" "${COMPONENT:?}" "${GITHUB_SHA:?}" "${GITHUB_OUTPUT:?}"
case "$COMPONENT" in api|web) ;; *) echo 'Unsupported image component' >&2; exit 1 ;; esac
[[ "$GITHUB_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid commit SHA' >&2; exit 1; }
version="${CUSTOM_TAG:-}"
if [[ "${GITHUB_REF_TYPE:-}" == tag ]]; then version="${GITHUB_REF_NAME:?}"; fi
if [[ -n "$version" && ! "$version" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,99}$ ]]; then
  echo 'Tag must be 1–100 letters, digits, underscores, periods or hyphens, and cannot start with a period or hyphen.' >&2
  exit 1
fi
{
  echo 'tags<<AUTOOPS_IMAGE_TAGS'
  printf '%s:%s-sha-%s\n' "$IMAGE_REPOSITORY" "$COMPONENT" "${GITHUB_SHA:0:12}"
  if [[ -n "$version" ]]; then printf '%s:%s-%s\n' "$IMAGE_REPOSITORY" "$COMPONENT" "$version"; fi
  echo 'AUTOOPS_IMAGE_TAGS'
} >> "$GITHUB_OUTPUT"
