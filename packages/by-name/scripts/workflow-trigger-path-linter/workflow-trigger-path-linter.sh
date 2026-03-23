#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail
shopt -s nullglob globstar

if [[ $# -eq 0 ]]; then
  echo "Usage: $0 <workflow-file>..." >&2
  exit 1
fi

repo_root=$(git rev-parse --show-toplevel)

errors=0

for workflow_file in "$@"; do
  # Extract paths from on.push.paths and on.pull_request.paths triggers.
  # yq outputs one path per line.
  paths=$(yq eval '
    [.on.push.paths // [], .on.pull_request.paths // []] | flatten | .[]
  ' "$workflow_file" 2>/dev/null) || continue

  if [[ -z $paths ]]; then
    continue
  fi

  while IFS= read -r path; do
    # Skip negated paths (exclusions).
    if [[ $path == "!"* ]]; then
      continue
    fi

    # Skip the catch-all glob.
    if [[ $path == "**" ]]; then
      continue
    fi

    # For paths ending with /** where the prefix is a literal directory, check it exists.
    if [[ $path == *"/**" ]]; then
      dir="${path%/\*\*}"
      if [[ $dir != *"*"* && $dir != *"?"* && $dir != *"["* ]]; then
        if [[ ! -d "${repo_root}/${dir}" ]]; then
          echo "error: ${workflow_file}: trigger path '${path}': directory '${dir}' does not exist" >&2
          errors=$((errors + 1))
        fi
        continue
      fi
    fi

    # For glob patterns, use bash globbing to check for at least one match.
    if [[ $path == *"*"* || $path == *"?"* || $path == *"["* ]]; then
      # shellcheck disable=SC2206 # intentional globbing
      matches=("${repo_root}"/${path})
      if [[ ${#matches[@]} -eq 0 ]]; then
        echo "error: ${workflow_file}: trigger path '${path}' does not match any files" >&2
        errors=$((errors + 1))
      fi
      continue
    fi

    # Literal path — check if it exists.
    if [[ ! -e "${repo_root}/${path}" ]]; then
      echo "error: ${workflow_file}: trigger path '${path}' does not exist" >&2
      errors=$((errors + 1))
    fi
  done <<<"$paths"
done

if [[ $errors -gt 0 ]]; then
  echo "${errors} trigger path(s) not found" >&2
  exit 1
fi
