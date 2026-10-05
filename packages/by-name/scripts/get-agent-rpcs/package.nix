# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  scripts,
  kata,
  kubectl,
  yq-go,
  gnugrep,
  coreutils,
}:

writeShellApplication {
  name = "get-agent-rpcs";
  runtimeInputs = [
    scripts.normalize-agent-rpcs
    kata.genpolicy
    kubectl
    yq-go
    gnugrep
    coreutils
  ];
  runtimeEnv = {
    DEBUGGER_YAML = ./debugger.yml;
  };
  text = builtins.readFile ./get-agent-rpcs.sh;
}
