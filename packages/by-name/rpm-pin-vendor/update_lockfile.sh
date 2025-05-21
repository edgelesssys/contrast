#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail
shopt -s inherit_errexit

DNF=${DNF:-dnf4}
DNFCONFIG=${DNFCONFIG:-@DNFCONFIG@}
REPOSDIR=${REPOSDIR:-@REPOSDIR@}
PACKAGESET=("$@")
OUT=${OUT:-$(mktemp -d)}

dnfroot=$(mktemp -d)
trap 'rm -rf $dnfroot' EXIT

cachedir="$dnfroot/cache"
mkdir -p "$cachedir"
varsdir="$dnfroot/vars"
mkdir -p "$varsdir"
persistdir="$dnfroot/persist"
mkdir -p "$persistdir"
logdir="$dnfroot/log"
mkdir -p "$logdir"

echo "Installing RPMs: ${PACKAGESET[*]}" >&2
echo "Writing rpms to $OUT" >&2

function urllist() {
  # shellcheck disable=SC2046
  fakeroot "$DNF" download \
    --verbose \
    --assumeyes \
    --config "${DNFCONFIG}" \
    --releasever=2.0 \
    --arch=x86_64 \
    "--installroot=$OUT" \
    "--setopt=reposdir=${REPOSDIR}" \
    "--setopt=cachedir=$cachedir" \
    "--setopt=varsdir=$varsdir" \
    "--setopt=persistdir=$persistdir" \
    "--setopt=logdir=$logdir" \
    --setopt=check_config_file_age=0 \
    --disableplugin='*' \
    --enableplugin=download \
    --urls \
    --resolve \
    --alldeps \
    "${PACKAGESET[@]}" >"$OUT/packages.txt"
}

function download() {
  mkdir -p "$OUT/download"
  echo "[]" >"$OUT/index.json"
  while read -r line; do
    if [[ $line != https* ]]; then
      continue
    fi
    wget -qP "$OUT/download" "$line"
    local filename
    filename=$(basename "$line")
    hash=$(nix hash file --sri --type sha256 "$OUT/download/$filename")
    jq ". += [{\"url\": \"$line\", \"hash\": \"$hash\"}]" "$OUT/index.json" >"$OUT/index.json.tmp"
    mv "$OUT/index.json.tmp" "$OUT/index.json"
  done <"$OUT/packages.txt"
  jq '. | sort_by(.url)' "$OUT/index.json" >"$OUT/index.json.tmp"
  mv "$OUT/index.json.tmp" "$OUT/index.json"
  jq . "$OUT/index.json"
}

urllist
download
