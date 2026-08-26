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
    hash = "sha256-Qc2LDt9AI//40T68ZzQT3jO3/QwL77P3z8M+eC3fKz4=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };

  rust = fetchzip {
    url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.zst";
    hash = "sha256-BOPN8C7MfzJnOF/WnvqdDhvkcWjjM1qjy8lv0vQUdIc=";
    stripRoot = false;
    nativeBuildInputs = [ zstd ];

    passthru.version = version;
    passthru.updateScript = ./update.sh;
  };
}
