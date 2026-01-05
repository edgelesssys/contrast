# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  inotify-tools,
  coreutils,
  findutils,
}:

writeShellApplication {
  name = "collect-logs";
  runtimeInputs = [
    inotify-tools
    coreutils
    findutils
  ];
  text = builtins.readFile ./script.sh;
}
