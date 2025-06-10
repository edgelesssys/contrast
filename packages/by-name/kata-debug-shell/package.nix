# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ writeShellApplication }:

writeShellApplication {
  name = "kata-debug-shell";
  # Don't provide runtimeInputs here, we want to use what's installed on the target system.
  text = builtins.readFile ./kata-debug-shell.sh;
}
