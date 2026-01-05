#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

mkdir /export

# collect all logs that may have been missed during startup
find /logs -name "*.log" |
  while read -r file; do
    if [[ -f $file && $file == *"$POD_NAMESPACE"* ]]; then
      mkdir -p "/export$(dirname "$file")"
      tail --follow=name "$file" >"/export$file" &
    fi
  done
inotifywait -m /logs -r -e create -e moved_to |
  while read -r path _action file; do
    filepath="$path$file"
    if [[ -f $filepath && $filepath == *"$POD_NAMESPACE"* ]]; then
      mkdir -p "/export$path"
      tail --follow=name "$filepath" >"/export$filepath" &
    fi
  done
