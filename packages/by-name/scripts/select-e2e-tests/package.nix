# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  coreutils,
  git,
  gnugrep,
  gnused,
  go,
  jq,
  nix,
}:

writeShellApplication {
  name = "select-e2e-tests";
  text = builtins.readFile ./select-e2e-tests.sh;
  runtimeInputs = [
    coreutils
    git
    gnugrep
    gnused
    go
    jq
    nix
  ];
}
