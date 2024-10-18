# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  buildGoModule,
  fetchFromGitHub,
  pkg-config,
  libvirt,
  writeShellApplication,
  gnugrep,
  runCommand,

  # List of supported cloud providers
  builtinCloudProviders ? [
    "aws"
    "azure"
    "gcp"
    "ibmcloud"
    "vsphere"
    # "libvirt"
    # "docker"
  ],

  cloud-api-adaptor,
}:

let
  withLibvirt = lib.strings.elem "libvirt" builtinCloudProviders;
in

buildGoModule rec {
  pname = "cloud-api-adaptor";
  version = "0.9.0";

  src = fetchFromGitHub {
    owner = "confidential-containers";
    repo = "cloud-api-adaptor";
    rev = "v${version}";
    hash = "sha256-5tDG0sEiRAsb259lPui5ntR6DVHDdcXhb04UESJzHhE=";
  };

  patches = [
    # Make the process-user-data job measure the agent config file into PCR10 (which is otherwise
    # unused), so that it can be verified in an attestation report.
    # The CAA attestation story is not decided yet. This patch enables one possible solution for
    # Contrast.
    ./0001-measure-agent-config.toml-into-PCR-10.patch
    # Forward the expected policy hash as part of the agent-config.toml via instance user-data.
    # Not upstreamable, like the patch above.
    ./0002-set-policy-digest-in-agent-config.patch
  ];

  patchFlags = [ "-p3" ];

  sourceRoot = "${src.name}/src/cloud-api-adaptor";

  proxyVendor = true;
  vendorHash = "sha256-6FWMh2G5yM0QnhpfLS+fRfP6bpPtuGCeCvCNutog3YU=";

  nativeBuildInputs = lib.optional withLibvirt pkg-config;

  buildInputs = lib.optional withLibvirt libvirt;

  subPackages = [
    "cmd/cloud-api-adaptor"
    "cmd/agent-protocol-forwarder"
    "cmd/process-user-data"
  ];

  CGO_ENABLED = if withLibvirt then 1 else 0;

  tags = builtinCloudProviders;

  ldflags = [
    "-X 'github.com/confidential-containers/cloud-api-adaptor/src/cloud-api-adaptor/cmd.VERSION=${version}'"
  ];

  passthru = {
    kata-agent-clean = writeShellApplication {
      name = "kata-agent-clean";
      runtimeInputs = [ gnugrep ];
      text = builtins.readFile "${cloud-api-adaptor.src}/src/cloud-api-adaptor/podvm/files/usr/local/bin/kata-agent-clean";
    };

    default-policy = runCommand "default-policy" { } ''
      cp ${cloud-api-adaptor.src}/src/cloud-api-adaptor/podvm/files/etc/kata-opa/allow-all.rego $out
    '';

    entrypoint = writeShellApplication {
      name = "entrypoint";
      runtimeInputs = [ cloud-api-adaptor ];
      text = builtins.readFile "${cloud-api-adaptor.src}/src/cloud-api-adaptor/entrypoint.sh";
      bashOptions = [
        "pipefail"
      ];
      excludeShellChecks = [
        "SC2086"
        "SC2153"
      ];
    };
  };

  meta = {
    description = "Ability to create Kata pods using cloud provider APIs aka the peer-pods approach";
    homepage = "https://github.com/confidential-containers/cloud-api-adaptor";
    license = lib.licenses.asl20;
    mainProgram = "cloud-api-adaptor";
  };
}
