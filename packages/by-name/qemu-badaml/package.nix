# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  qemu-cc,
  runCommandLocal,
}:

runCommandLocal "qemu-badaml" { } ''
  cp -r ${qemu-cc} $out
  chmod +w $out/bin
  mv $out/bin/qemu-system-x86_64 $out/bin/qemu-system-x86_64-wrapped

  cat << 'EOF' > $out/bin/qemu-system-x86_64
  #!/usr/bin/env bash
  SCRIPT_PATH="$0"
  SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd)

  BINARY_PATH="$SCRIPT_DIR/qemu-system-x86_64-wrapped"
  AML_PATH="$SCRIPT_DIR/payload.aml"

  exec "$BINARY_PATH" "$@" -acpitable file="$AML_PATH"
  EOF

  chmod +x $out/bin/qemu-system-x86_64
''
