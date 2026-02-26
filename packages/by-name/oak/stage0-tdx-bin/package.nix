# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  fetchurl,
}:

fetchurl {
  pname = "stage0-tdx-bin";
  version = "d915570d";
  url = "https://contrast-public.s3.eu-central-1.amazonaws.com/hack/stage0_bin_tdx";
  hash = "sha256-oU3yTAj30g4ZqtJpweyFM4Mje5mnpByS2QiqGqRgys4=";
}
