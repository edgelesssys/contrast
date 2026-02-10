# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runtime,
  fetchzip,
  zstd,
}:

let
  inherit (runtime) version;
in
fetchzip {
  url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.zst";
  hash = "sha256-R7ZW7b5WY63qI2tLS8f7tFKiuTe8ynymj2bxWEHZXYg=";
  stripRoot = false;
  nativeBuildInputs = [ zstd ];

  passthru.version = version;
  passthru.updateScript = ./update.sh;
}
