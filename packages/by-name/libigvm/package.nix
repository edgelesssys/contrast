# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchFromGitHub,
  rustPlatform,
  rust-cbindgen,
  gettext,
}:

rustPlatform.buildRustPackage rec {
  pname = "libigvm";
  version = "0.3.3";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "igvm";
    rev = "igvm-v${version}";
    hash = "sha256-zBsvKcv9BSBEDXznn1MvXtiwZiAECg+7cUBCdbjUw7Y=";
  };
  postPatch = ''
    ln -s ${./Cargo.lock} Cargo.lock
  '';

  cargoLock = {
    lockFile = ./Cargo.lock;
  };

  cargoHash = "sha256-tbrTbutUs5aPSV+yE0IBUZAAytgmZV7Eqxia7g+9zRR=";

  cargoBuildFlags = "-p igvm";
  buildFeatures = [ "igvm-c" ];

  nativeBuildInputs = [
    rust-cbindgen
    gettext
  ];

  postInstall = ''
    # TODO(freax13): Vendored from https://github.com/microsoft/igvm/blob/ec710cbecdf5f41ddd772f79ae01a749c1478f8c/igvm_c/Makefile

    cbindgen -q -c igvm_c/cbindgen_igvm.toml igvm -o "igvm_c/include/igvm.h"
    cbindgen -q -c igvm_c/cbindgen_igvm_defs.toml igvm_defs -o "igvm_c/include/igvm_defs.h"
    bash igvm_c/scripts/post_process.sh "igvm_c/include"

    mkdir -p $out/lib/pkgconfig
    mkdir -p $out/include/igvm
    install -m 644 igvm_c/include/* $out/include/igvm
    VERSION=$version PREFIX=$out envsubst '$$VERSION $$PREFIX' \
    			< igvm_c/igvm.pc.in \
    			> $out/lib/pkgconfig/igvm.pc
  '';

  meta = {
    changelog = "https://github.com/coconut-svsm/svsm/releases/tag/${version}";
    homepage = "https://github.com/coconut-svsm/svsm";
    mainProgram = "igvmmeasure";
    license = lib.licenses.mit;
  };
}
