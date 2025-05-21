# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  python3,
  writeTextFile,
}:

python3.pkgs.buildPythonApplication {
  pname = "igvm-signing-keygen";
  version = "1.5.0";
  pyproject = true;

  src = ./.;

  propagatedBuildInputs = with python3.pkgs; [
    ecdsa
    setuptools
  ];

  passthru.snakeoilPem = writeTextFile {
    name = "snakeoil.pem";
    text = builtins.readFile ./snakeoil.pem;
  };

  meta = {
    description = "Signing key for IGVM ID block";
    mainProgram = "gen_signing_pem";
    platforms = lib.platforms.all;
  };
}
