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
  hash = "sha256-esYl9anGN4KJ3qsxzpvGhfZTFPxJK8tnOyxmv6TGPmg=";
  stripRoot = false;
  nativeBuildInputs = [ zstd ];

  passthru.version = version;
  passthru.updateScript = ./update.sh;
}
