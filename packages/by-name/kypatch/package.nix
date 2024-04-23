# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ writeShellApplication, yq-go }:

writeShellApplication {
  name = "kypatch";
  runtimeInputs = [ yq-go ];
  text = builtins.readFile ./kypatch.sh;
}
