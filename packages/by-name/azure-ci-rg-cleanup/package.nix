# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  azure-cli,
}:

writeShellApplication {
  name = "azure-ci-rg-cleanup";
  runtimeInputs = [ azure-cli ];
  text = builtins.readFile ./cleanup.sh;
}
