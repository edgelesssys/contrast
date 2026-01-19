# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  pkgs,
  lib,
  writeShellApplication,
  kubectl,
  inotify-tools,
  fswatch,
}:

# Usage: get-logs [start | download] $namespaceFile
let
  inherit (pkgs.stdenv.hostPlatform) isDarwin;
  watcher =
    if isDarwin then
      ''
        fswatch -1 --event Removed --event Renamed "$namespace_file" >/dev/null
      ''
    else
      ''
        local dir base
        dir="$(dirname -- "$namespace_file")"
        base="$(basename -- "$namespace_file")"

        while :; do
          inotifywait -q -e delete,moved_from --format '%f' "$dir" |
            grep -qx "$base" && break
        done
      '';

  processedScript = lib.replaceStrings [ "@watcher@" ] [ watcher ] (builtins.readFile ./get-logs.sh);
in
writeShellApplication {
  name = "get-logs";

  runtimeInputs = [
    kubectl
  ]
  ++ (if isDarwin then [ fswatch ] else [ inotify-tools ]);

  text = processedScript;
}
