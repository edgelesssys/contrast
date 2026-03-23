# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# QEMU debug wrapper. Wraps the real qemu-system-x86_64 binary with
# configurable extra features:
#
#   withACPITable      - When true, injects payload.aml (expected next to
#                        the wrapper) as an extra ACPI table via -acpitable.
#
#   withSerialLog      - When true, captures serial output to timestamped
#                        log files under <runtime-dir>/logs/ on the node.

{
  lib,
  qemu-cc,
  runCommandLocal,
  shellcheck-minimal,
  writeTextFile,
  withACPITable ? false,
  withSerialLog ? false,
}:

let
  acpiTableSnippet = ''
    EXTRA_ARGS+=(-acpitable "file=$SCRIPT_DIR/payload.aml")
  '';

  serialLogSnippet = ''
    RUNTIME_DIR=""
    SANDBOX_ID="unknown"
    for arg in "$@"; do
      if [[ -z "$RUNTIME_DIR" && "$arg" == /opt/edgeless/*/share/* ]]; then
        RUNTIME_DIR="''${arg%/share/*}"
      fi
      if [[ "$arg" == sandbox-* ]]; then
        SANDBOX_ID="''${arg%%,*}"
        SANDBOX_ID="''${SANDBOX_ID#sandbox-}"
      fi
    done

    SHORT_ID="''${SANDBOX_ID:0:12}"
    TIMESTAMP=$(date +%Y%m%d-%H%M%S)

    LOG_DIR="''${RUNTIME_DIR:-/tmp}/logs"
    mkdir -p "$LOG_DIR"
    EXTRA_ARGS+=(-serial "file:$LOG_DIR/''${TIMESTAMP}_''${SHORT_ID}.log")
  '';

  wrapper = writeTextFile {
    name = "qemu-system-x86_64";
    executable = true;
    destination = "/bin/qemu-system-x86_64";
    text = ''
      #!/usr/bin/env bash

      SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

      EXTRA_ARGS=()

    ''
    + lib.optionalString withACPITable acpiTableSnippet
    + lib.optionalString withSerialLog serialLogSnippet
    + ''

      exec "$SCRIPT_DIR/qemu-system-x86_64-wrapped" "$@" "''${EXTRA_ARGS[@]}"
    '';
    checkPhase = ''
      ${lib.getExe shellcheck-minimal} "$target"
    '';
  };
in

runCommandLocal "qemu-wrapped" { } ''
  cp -r ${qemu-cc} $out
  chmod +w $out/bin

  mv $out/bin/qemu-system-x86_64 $out/bin/qemu-system-x86_64-wrapped
  cp ${wrapper}/bin/qemu-system-x86_64 $out/bin/qemu-system-x86_64
''
