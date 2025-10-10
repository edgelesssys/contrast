# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runtime,
  pkgs,
  stdenv,
  fetchurl,
}:

let
  inherit (runtime) version;
in

stdenv.mkDerivation {
  name = "src";
  src = fetchurl {
    url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.zst";
    hash = "sha256-A8qH1gKcuzoQPWqnTzsg6zn4qLF5Xk4bLY0u4gsr4Ag=";
  };

  nativeBuildInputs = [ pkgs.zstd ];

  unpackPhase = ''
    mkdir -p $out
    tar --zstd -xvf $src -C $out
  '';

  dontBuild = true;

  passthru.version = version;
  passthru.updateScript = ./update.sh;
}
