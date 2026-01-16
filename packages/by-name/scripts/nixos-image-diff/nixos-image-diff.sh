#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

size_human() {
  local size_bytes="$1"
  local sign=""
  local abs="$size_bytes"

  if [[ $size_bytes -lt 0 ]]; then
    sign="-"
    abs=$((-size_bytes))
  fi
  if [[ $abs -lt 1024 ]]; then
    echo "${sign}${abs} B"
  elif [[ $abs -lt $((1024 * 1024)) ]]; then
    echo "${sign}$((abs / 1024)) KB"
  elif [[ $abs -lt $((1024 * 1024 * 1024)) ]]; then
    echo "${sign}$((abs / (1024 * 1024))) MB"
  else
    echo "${sign}$((abs / (1024 * 1024 * 1024))) GB"
  fi
}

get_files_size() {
  local dir="$1"
  find "$dir" | while IFS= read -r path; do
    rpath=$(realpath "$path")
    if [[ ! -f $rpath ]]; then
      continue
    fi
    size=$(stat -c%s "$rpath")
    rel=${path#"$dir"/}
    echo "$rel $size"
  done
}

diff_files_size() {
  local prev_sizes="$1"
  local head_sizes="$2"

  local w_file
  w_file=$(printf '%s\n' "$prev_sizes" | awk '{print length($1)}' | sort -nr | head -n1)

  echo "$prev_sizes" | while IFS= read -r line; do
    file=$(echo "$line" | awk '{print $1}')
    size_prev=$(echo "$line" | awk '{print $2}')
    size_prev_h=$(size_human "$size_prev")
    size_head=$(echo "$head_sizes" | grep "^$file " | awk '{print $2}' || echo "0")
    size_head_h=$(size_human "$size_head")
    if [[ $size_prev != "$size_head" ]]; then
      diff=$((size_head - size_prev))
      diff_h=$(size_human "$diff")
      printf "%-*s  %7s -> %7s  (diff: %7s)\n" \
        "$w_file" "$file" "$size_prev_h" "$size_head_h" "$diff_h"
    else
      printf "%-*s  %7s             (no change)\n" \
        "$w_file" "$file" "$size_prev_h"
    fi
  done
}

image_attribute="${1:-}"
if [[ -z ${image_attribute} ]]; then
  echo "Usage: $0 <image-attribute>"
  exit 1
fi
head="${2:-HEAD}"
prev="${3:-HEAD~1}"

repo_root="$(git rev-parse --show-toplevel)"
tmpdir="$(mktemp -d -t git-worktree.XXXXXX)"
worktree="$tmpdir/wt"

cleanup() {
  git -C "$repo_root" worktree remove --force "$worktree" 2>/dev/null || true
  rm -rf "$tmpdir" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

git -C "$repo_root" worktree add --quiet --detach "$worktree" "$head"
pushd "$worktree" >/dev/null

commit_head=$(git log -1 --format='%h %s' HEAD)
image_path_head=$(nix build --print-out-paths "$image_attribute")
file_sizes_head=$(get_files_size "$image_path_head")
toplevel_path_head=$(nix build --print-out-paths "$image_attribute".toplevel)
toplevel_head=$(nix-store --query --requisites "$toplevel_path_head")

git checkout "$prev" --quiet
commit_prev=$(git log -1 --format='%h %s' HEAD)
image_path_prev=$(nix build --print-out-paths "$image_attribute")
file_sizes_prev=$(get_files_size "$image_path_prev")
toplevel_path_prev=$(nix build --print-out-paths "$image_attribute".toplevel)
toplevel_prev=$(nix-store --query --requisites "$toplevel_path_prev")

popd >/dev/null

echo "Comparing $commit_prev"
echo "       -> $commit_head"
echo
echo "Size diff in store path:"
echo
diff_files_size "$file_sizes_prev" "$file_sizes_head" || true
echo
echo "Toplevel diff (ignoring store paths hashes):"
echo
git diff --color=always --no-index \
  <(cut -d- -f2- <<<"$toplevel_prev" | sort -u) \
  <(cut -d- -f2- <<<"$toplevel_head" | sort -u) |
  grep --text --color=always -E $'^(\x1b\\[[0-9;]*m)*[+-]' |
  grep --text --color=always -E -v $'[ab]/dev/fd' || true
