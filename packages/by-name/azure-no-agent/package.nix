# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  busybox,
  curlMinimal,
}:

writeShellApplication {
  name = "azure-no-agent";

  runtimeInputs = [
    busybox
    curlMinimal
  ];

  text = builtins.readFile ./no-agent.sh;

  meta = {
    mainProgram = "azure-no-agent";
    homepage = "https://learn.microsoft.com/en-us/azure/virtual-machines/linux/no-agent";
  };
}
