# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  iproute2,
  iptables,
  sysctl,
  gawk,
}:

writeShellApplication {
  name = "peerpod-imds-nat";

  runtimeInputs = [
    iproute2
    iptables
    sysctl
    gawk
  ];

  text = builtins.readFile ./setup-nat-for-imds.sh;

  meta = {
    mainProgram = "peerpod-imds-nat";
    homepage = "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/usr/local/bin/setup-nat-for-imds.sh";
  };
}
