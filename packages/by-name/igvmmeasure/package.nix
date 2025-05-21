# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  fetchFromGitHub,
  rustPlatform,
}:

rustPlatform.buildRustPackage rec {
  pname = "igvmmeasure";
  version = "0.1.0-unstable-2024-08-19";

  src = fetchFromGitHub {
    owner = "coconut-svsm";
    repo = "svsm";
    # TODO(malt3): Use a released version once available.
    rev = "aa4936afbfae394a0b7404e5080863e5dae9473d";
    hash = "sha256-QHVdHVhQJhrmE8Vjg4Xw2HJmm2l9hMB0iL1KHVlh1iw=";
  };

  buildAndTestSubdir = "igvmmeasure";

  cargoLock = {
    lockFile = "${src}/Cargo.lock";
    outputHashes = {
      "packit-0.1.1" = "sha256-jrH0y1ebpUilV+nyv/kLQzcZP1lMW1fVCQo35tz5Vhs=";
    };
  };

  meta = {
    changelog = "https://github.com/coconut-svsm/svsm/releases/tag/${version}";
    homepage = "https://github.com/coconut-svsm/svsm";
    mainProgram = "igvmmeasure";
    license = lib.licenses.mit;
  };
}
