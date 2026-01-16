# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  fetchurl,
}:

fetchurl {
  pname = "stage0-bin";
  version = "d915570d";
  url = "https://contrast-public.s3.eu-central-1.amazonaws.com/hack/stage0_bin";
  hash = "sha256-0NqgKgegnwPg8503E/Dt/UsBK2HWwifJQnkcfZEIG3g=";
}
