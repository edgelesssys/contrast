# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  git,
  yq-go,
}:

writeShellApplication {
  name = "workflow-trigger-path-linter";
  text = builtins.readFile ./workflow-trigger-path-linter.sh;
  runtimeInputs = [
    git
    yq-go
  ];
}
