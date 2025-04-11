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
  iptables,
  iproute2,
  sysctl,
  gawk,
  runCommand,
  applyPatches,
  makeWrapper,

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

buildGoModule (finalAttrs: {
  pname = "cloud-api-adaptor";
  version = "0.10.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "confidential-containers";
      repo = "cloud-api-adaptor";
      rev = "v${finalAttrs.version}";
      hash = "sha256-OaSIO26nlkeI2olSx0o8xdwhLMZ8eH753pUbyHypI+E=";
    };

    patches = [
      # This fixes a route setting problem we see with our NixOS image that
      # does not seem to occur with the upstream image.
      # TODO(burgerdev): upstream
      ./0001-netops-replace-routes-instead-of-adding-them.patch
    ];
  };

  sourceRoot = "${finalAttrs.src.name}/src/cloud-api-adaptor";

  proxyVendor = true;
  vendorHash = "sha256-FsckYZAiBfTEp25+dDNqPpB/550NqeEsutWC34s+GmE=";

  nativeBuildInputs = [ makeWrapper ] ++ lib.optional withLibvirt pkg-config;

  buildInputs = lib.optional withLibvirt libvirt;

  subPackages = [
    "cmd/cloud-api-adaptor"
    "cmd/agent-protocol-forwarder"
    "cmd/process-user-data"
  ];

  env.CGO_ENABLED = if withLibvirt then 1 else 0;

  tags = builtinCloudProviders;

  ldflags = [
    "-X 'github.com/confidential-containers/cloud-api-adaptor/src/cloud-api-adaptor/cmd.VERSION=${finalAttrs.version}'"
  ];

  postInstall = ''
    wrapProgram $out/bin/agent-protocol-forwarder --prefix PATH : ${lib.makeBinPath [ iptables ]}
  '';

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

    setup-nat-for-imds = writeShellApplication {
      name = "setup-nat-for-imds";
      runtimeInputs = [
        iproute2
        iptables
        sysctl
        gawk
      ];
      # TODO(burgerdev): generalize for all link-local IPs and investigate routing simplification
      text = builtins.readFile "${cloud-api-adaptor.src}/src/cloud-api-adaptor/podvm/files/usr/local/bin/setup-nat-for-imds.sh";
      meta = {
        mainProgram = "setup-nat-for-imds";
        homepage = "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/usr/local/bin/setup-nat-for-imds.sh";
      };
    };
  };

  meta = {
    description = "Ability to create Kata pods using cloud provider APIs aka the peer-pods approach";
    homepage = "https://github.com/confidential-containers/cloud-api-adaptor";
    license = lib.licenses.asl20;
    mainProgram = "cloud-api-adaptor";
  };
})
