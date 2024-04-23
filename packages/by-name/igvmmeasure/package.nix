# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, fetchFromGitHub
, rustPlatform
, cmake
}:

rustPlatform.buildRustPackage rec {
  pname = "igvmmeasure";
  version = "0.1.0-unstable-2024-03-25";

  src = fetchFromGitHub {
    owner = "coconut-svsm";
    repo = "svsm";
    # TODO(malt3): Use a released version once available.
    rev = "509bc7ca181af6981a1d50fb1cc46320553a4370";
    hash = "sha256-kA5ZI+8RpG2swApPLZsWJdcUu09sUBpdSAQf844XVuU=";
  };

  cargoBuildFlags = "-p igvmmeasure";

  cargoLock = {
    lockFile = "${src}/Cargo.lock";
    outputHashes = {
      "packit-0.1.1" = "sha256-BLVpKYjrqTwEAPgL7V1xwMnmNn4B8bA38GSmrry0GIM=";
    };
  };

  meta = {
    changelog = "https://github.com/coconut-svsm/svsm/releases/tag/${version}";
    homepage = "https://github.com/coconut-svsm/svsm";
    mainProgram = "igvmmeasure";
    license = lib.licenses.mit;
  };
}
