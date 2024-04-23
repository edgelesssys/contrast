#!/user/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

function usage() {
  cat <<EOF >&2
Usage: $0 <images|namespace> [options] <target-path>

Options:
  --replace <current> <new>     Replace current value with new value
  --help                        Show this help message
EOF
}

function printReplaces() {
  echo "Replacements:" >&2
  for replace in "${replaces[@]}"; do
    currentImage=${replace%% *}
    newImage=${replace##* }
    echo "  $currentImage => $newImage" >&2
  done
}

function mapTypeToPaths() {
  local -n outPaths=$1
  local type=$2

  case $type in
  images)
    # shellcheck disable=SC2034
    outPaths=(
      ".spec.containers[].image"
      ".spec.template.spec.containers[].image"
      ".spec.template.spec.initContainers[].image"
    )
    ;;
  namespace)
    # shellcheck disable=SC2034
    outPaths=(
      ".metadata.namespace"
    )
    ;;
  *)
    echo "Unknown replace target type $type" >&2
    exit 1
    ;;
  esac
}

function extraSteps() {
  local type=$1
  local file=$2
  local replace=$3

  current=${replace%% *}
  new=${replace##* }

  case $type in
  namespace)
    # Rename metadata.name if kind is Namespace
    yq -i "\
      with(
        select(.kind == \"Namespace\") | \
        select(.metadata.name | contains(\"${current}\")); \
        .metadata.name |= sub(\"${current}\", \"${new}\") \
      )" "$file"
    ;;
  esac
}

function patchFile() {
  local type=$1
  local file=$2
  shift 2
  local replaces=("$@")

  echo "Patching file $file" >&2

  for replace in "${replaces[@]}"; do
    current=${replace%% *}
    new=${replace##* }

    local paths
    mapTypeToPaths paths "$type"

    for p in "${paths[@]}"; do
      yq -i "\
        with(select(${p} | contains(\"${current}\")); \
          ${p} |= sub(\"${current}\", \"${new}\") \
        )" "$file"

    done

    extraSteps "$type" "$file" "$replace"
  done
}

function patchRecursive() {
  local type=$1
  local dir=$2
  shift 2
  local replaces=("$@")

  find "$dir" \
    -type f \
    -name '*.yaml' -o \
    -name '*.yml' | while IFS= read -r file; do
    patchFile "$type" "$file" "${replaces[@]}"
  done
}

function main() {
  positionalArgs=()
  replaces=()
  while [[ $# -gt 0 ]]; do
    case $1 in
    --replace)
      replaces+=("$2 $3")
      shift 3 # past flag, current, new
      ;;
    --help)
      usage
      exit 0
      ;;
    -*)
      echo "Unknown option $1" >&2
      exit 1
      ;;
    *)
      positionalArgs+=("$1") # save positional arg
      shift                  # past argument
      ;;
    esac
  done
  set -- "${positionalArgs[@]}" # restore positional parameters

  type=$1
  targetPath=$2

  printReplaces "${replaces[@]}"

  if [[ -d $targetPath ]]; then
    patchRecursive "$type" "$targetPath" "${replaces[@]}"
    exit 0
  fi
  patchFile "$type" "$targetPath" "${replaces[@]}"
}

main "$@"
