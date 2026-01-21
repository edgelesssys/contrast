# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  pkgs,
  lib,
  writeShellApplication,
}:

let
  inherit (pkgs.stdenv.hostPlatform) isDarwin;
  inherit (pkgs.stdenv.hostPlatform) isLinux;
in
writeShellApplication {
  name = "wait-for-port-listen";

  runtimeInputs = lib.optional isDarwin pkgs.lsof ++ lib.optional isLinux pkgs.iproute2;

  text = ''
      port=$1
      tries=15 # 3 seconds
      interval=0.2

      function listen-on-port() {
        ${
          if isDarwin then
            ''lsof -i :"$port" -sTCP:LISTEN -t''
          else
            ''ss --tcp --numeric --listening --no-header --ipv4 src ":$port"''
        }
      }

      while [[ "$tries" -gt 0 ]]; do
        if [[ -n $(listen-on-port) ]]; then
          exit 0
        fi
        sleep "$interval"
        tries=$((tries - 1))
      done

    echo "Port $port did not reach state LISTENING" >&2
    exit 1
  '';
}
