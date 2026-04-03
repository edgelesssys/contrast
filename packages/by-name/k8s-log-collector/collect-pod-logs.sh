#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

mkdir /export

# collect all logs that may have been missed during startup
find /logs -type f -name '*.log' -path "*$POD_NAMESPACE*" -exec sh -c '
  mkdir -p "/export$(dirname "$1")"
  tail --follow=name "$1" >"/export$1" &
' sh {} \;

inotifywait -m -r -e create -e moved_to --format '%w%f' /logs |
  stdbuf -oL grep "$POD_NAMESPACE" |
  while IFS= read -r filepath; do
    [[ -f $filepath && $filepath == *.log ]] || continue

    mkdir -p "/export$(dirname "$filepath")"
    tail --follow=name "$filepath" >"/export$filepath" &
  done
