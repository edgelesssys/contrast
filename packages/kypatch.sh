#!/user/bin/env bash

set -euo pipefail

function usage() {
  cat <<EOF >&2
Usage: $0 [options] <target-path>

Options:
  --replace <current-image> <new-image>     Replace current-image with new-image
  --help                                    Show this help message
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
  *)
    echo "Unknown replace target type $type" >&2
    exit 1
    ;;
  esac
}

function patchFile() {
  local type=$1
  local file=$2
  shift
  local replaces=("$@")

  echo "Patching file $file" >&2

  for replace in "${replaces[@]}"; do
    currentImage=${replace%% *}
    newImage=${replace##* }

    local paths
    mapTypeToPaths paths "$type"

    for p in "${paths[@]}"; do
      yq -i "\
        with(select(${p} | contains(\"${currentImage}\")); \
          ${p} |= sub(\"${currentImage}\", \"${newImage}\") \
        )" "$file"
    done
  done
}

function patchRecursive() {
  local type=$1
  local dir=$2
  shift
  local replaces=("$@")

  find "$dir" \
    -type f \
    -name '*.yaml' -o \
    -name '*.yml' | while IFS= read -r file; do
    patchFile "$file" "${replaces[@]}"
  done
}

function main() {
  positionalArgs=()
  replaces=()
  while [[ $# -gt 0 ]]; do
    case $1 in
    --replace)
      replaces+=("$2 $3")
      shift # past argument
      shift # past value
      shift # past value
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
