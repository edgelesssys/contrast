# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  runc,
  yq-go,
  stdenv,
  musl,
}:

stdenv.mkDerivation {
  name = "pause-bundle";

  nativeBuildInputs = [
    runc
    yq-go
    musl
  ];

  dontUnpack = true;

  buildPhase = ''
    runHook preBuild

    mkdir -p $out/pause_bundle/rootfs

    cat <<EOF > pause.c
    #include <unistd.h>
    void main() { pause(); }
    EOF
    musl-gcc -static -o $out/pause_bundle/rootfs/pause pause.c

    runc spec
    yq -i '.process.args[0] = "/pause"' config.json
    mv config.json $out/pause_bundle/config.json

    runHook postBuild
  '';
}
