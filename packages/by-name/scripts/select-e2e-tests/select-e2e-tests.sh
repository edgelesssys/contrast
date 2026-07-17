#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

# get e2e test names from directories under e2e/ that hold a Go test file.
list_test_names() {
  local d name tf
  shopt -s nullglob
  for d in "$root"/e2e/*/; do
    name=${d%/}
    name=${name##*/}
    case $name in
    internal | regression | genpolicy-unsupported | release) continue ;;
    esac
    tf=("$d"*_test.go)
    [[ ${#tf[@]} -gt 0 ]] && printf '%s\n' "$name"
  done
  shopt -u nullglob
}

# test-if directives (path:... / nix:...) declared in test source.
directives_of() {
  local files
  shopt -s nullglob
  files=("$root/e2e/$1"/*.go)
  shopt -u nullglob
  [[ ${#files[@]} -gt 0 ]] || return 0
  { grep -hoE '//[[:space:]]*test-if:[[:space:]]*(path|nix):[^[:space:]]+' "${files[@]}" || true; } |
    sed -E 's|.*test-if:[[:space:]]*||'
}

declare -A selected=()
changed=()

if [[ ${1:-} == all ]]; then
  while IFS= read -r name; do selected[$name]=1; done < <(list_test_names)
else
  base=${1:?usage: select-e2e-tests <base-ref> [head-ref] | all | check}
  head=${2:-HEAD}
  base_sha=$(git rev-parse "$base")
  head_sha=$(git rev-parse "$head")
  mapfile -t changed < <(git diff --name-only "$base_sha" "$head_sha")

  path_matches() {
    local p=$1 f
    for f in "${changed[@]}"; do
      [[ $f == "$p" || $f == "$p"/* ]] && return 0
    done
    return 1
  }

  # Diff each referenced nix: artifact once.
  declare -A nix_attrs=() nix_changed=()
  while IFS= read -r name; do
    while IFS= read -r d; do
      [[ $d == nix:* ]] && nix_attrs[${d#nix:}]=1
    done < <(directives_of "$name")
  done < <(list_test_names)
  for attr in "${!nix_attrs[@]}"; do
    b=$(nix eval --raw "git+file://$root?rev=$base_sha#base.$attr.drvPath" 2>/dev/null || true)
    h=$(nix eval --raw "git+file://$root?rev=$head_sha#base.$attr.drvPath" 2>/dev/null || true)
    if [[ -z $b || -z $h ]]; then
      printf 'select-e2e-tests: warning: cannot evaluate nix:%s, assuming changed\n' "$attr" >&2
      nix_changed[$attr]=1
    elif [[ $b != "$h" ]]; then
      nix_changed[$attr]=1
    fi
  done

  while IFS= read -r name; do
    # Always run: openssl, gpu, and any test whose own e2e/<name>/ changed.
    if [[ $name == openssl || $name == gpu ]] || path_matches "e2e/$name"; then
      selected[$name]=1
      continue
    fi
    while IFS= read -r d; do
      case $d in
      path:*) path_matches "${d#path:}" && {
        selected[$name]=1
        break
      } ;;
      nix:*) [[ -n ${nix_changed[${d#nix:}]:-} ]] && {
        selected[$name]=1
        break
      } ;;
      esac
    done < <(directives_of "$name")
  done < <(list_test_names)
fi

# gpu and multi-runtime-class only do anything on GPU hosts, every other test runs on non-GPU.
for test in "${!selected[@]}"; do
  if [[ $test == gpu || $test == multi-runtime-class ]]; then
    printf '%s\tMetal-QEMU-SNP-GPU\tSNP-GPU\n' "$test"
    printf '%s\tMetal-QEMU-TDX-GPU\tTDX-GPU\n' "$test"
  else
    printf '%s\tMetal-QEMU-SNP\tSNP\n' "$test"
    printf '%s\tMetal-QEMU-TDX\tTDX\n' "$test"
  fi
done | LC_ALL=C sort | jq -Rsc '
  [ split("\n")[] | select(length > 0) | split("\t")
    | { test: .[0], platform: .[1], runner: .[2], "self-hosted": true } ]
'
