# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  kata,
  snp-id-block-generator,
}:

stdenvNoCC.mkDerivation {
  name = "snp-id-block";

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    ${lib.getExe snp-id-block-generator} \
      --launch-digest ${kata.snp-launch-digest}/milan.hex \
      --id-block-out $out/id-block-milan.base64 \
      --id-auth-out $out/id-auth-milan.base64
    ${lib.getExe snp-id-block-generator} \
      --launch-digest ${kata.snp-launch-digest}/genoa.hex \
      --id-block-out $out/id-block-genoa.base64 \
      --id-auth-out $out/id-auth-genoa.base64
  '';
}
