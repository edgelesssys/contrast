{
  qemu-cc,
  runCommandLocal,
  makeWrapper,
}:

runCommandLocal "qemu-badaml"
  {
    nativeBuildInputs = [ makeWrapper ];
  }
  ''
    cp -r ${qemu-cc} $out
    chmod +w $out/bin
    mv $out/bin/qemu-system-x86_64 $out/bin/qemu-system-x86_64-wrapped

    echo '#!/usr/bin/env bash
    SCRIPT_PATH="$0"
    SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd)

    BINARY_PATH="$SCRIPT_DIR/qemu-system-x86_64-wrapped"
    AML_PATH="$SCRIPT_DIR/payload.aml"

    exec "$BINARY_PATH" "$@" -acpitable file="$AML_PATH"' > $out/bin/qemu-system-x86_64

    chmod +x $out/bin/qemu-system-x86_64
  ''
