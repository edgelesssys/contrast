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
    hash = "sha256-FdPv/C78T2q2hi/FtNikeHiDFsIVmQLadw8hrZx3TPE=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };

  rust = fetchzip {
    url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.zst";
    hash = "sha256-God5+2kzpY+qyU6U302NTwuaThcWT/ssXZXA/JzCFck=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };
}
