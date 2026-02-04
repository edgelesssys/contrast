# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  fetchurl,
  runCommand,
  stage0-bin,
}:

fetchurl {
  pname = "stage0-bin";
  version = "dd1f39d06";
  url = "https://contrast-public.s3.eu-central-1.amazonaws.com/hack/stage0_bin_dd1f39d06";
  hash = "sha256-nRQutWB3LBFiHZ8kDsf83XvVll5GN7RR3djvP3ye8ok=";
  passthru.ovmf-imposter = runCommand "${stage0-bin.name}-ovmf-imposter" { } ''
    mkdir -p $out/FV
    cp ${stage0-bin} $out/FV/OVMF.fd
  '';
}
