# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  fetchurl,
}:

fetchurl {
  pname = "stage0-tdx-bin";
  version = "c5bd54d97";
  url = "https://contrast-public.s3.eu-central-1.amazonaws.com/hack/stage0_tdx_bin_c5bd54d97";
  hash = "sha256-B0k7u57cSmL16lu5wPlzAxwevTCpRYS7YKCJHc5TioU=";
}
