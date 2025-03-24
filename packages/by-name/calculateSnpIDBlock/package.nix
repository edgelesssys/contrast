# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  snp-id-block-generator,
}:

{
  snp-launch-digest,
  snp-guest-policy,
}:

stdenvNoCC.mkDerivation {
  name = "snp-id-block";

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    if [[ -f ${snp-launch-digest}/milan.hex ]]; then
      ${lib.getExe snp-id-block-generator} \
        --launch-digest ${snp-launch-digest}/milan.hex \
        --guest-policy ${snp-guest-policy} \
        --id-block-out $out/id-block-milan.base64 \
        --id-auth-out $out/id-auth-milan.base64 \
        --id-block-igvm-out $out/id-block-igvm-milan.json
    fi

    if [[ -f ${snp-launch-digest}/genoa.hex ]]; then
      ${lib.getExe snp-id-block-generator} \
        --launch-digest ${snp-launch-digest}/genoa.hex \
        --guest-policy ${snp-guest-policy} \
        --id-block-out $out/id-block-genoa.base64 \
        --id-auth-out $out/id-auth-genoa.base64 \
        --id-block-igvm-out $out/id-block-igvm-genoa.json
    fi
  '';
}
