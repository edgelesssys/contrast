# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  inotify-tools,
  coreutils,
  findutils,
  gnused,
  gnugrep,
}:

writeShellApplication {
  name = "collect-logs";
  runtimeInputs = [
    inotify-tools
    coreutils
    findutils
    gnugrep
    gnused
  ];
  text = builtins.readFile ./script.sh;
}
