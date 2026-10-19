# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runtime,
  fetchzip,
  zstd,
}:

rec {
  inherit (runtime) version;
  go = fetchzip {
    url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-go-static-${version}-amd64.tar.zst";
    hash = "sha256-r6jlSxLVhC+08+s7Y7EEY2B/kxGhNoa85MdYfK8wCy0=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };

  rust = fetchzip {
    url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.zst";
    hash = "sha256-kYB9/8+QHCiu/Gr2bLPqDC2riUAACxs959HIQdW2rTo=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };
}
