# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  rustPlatform,
  fetchFromGitHub,
  pkg-config,
  openssl,
  patchelf,
  withIGVM ? true,
  withSEVSNP ? true,
  withTDX ? false,
}:

rustPlatform.buildRustPackage rec {
  pname = "cloud-hypervisor";
  version = "38.0.72.3";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "cloud-hypervisor";
    rev = "refs/tags/msft/v${version}";
    hash = "sha256-14/OZhmiHDgJAZxMvD+vsepaB4gudThl+4nZyacywTI=";
  };

  cargoLock = {
    lockFile = "${src}/Cargo.lock";
    outputHashes = {
      "acpi_tables-0.1.0" = "sha256-syDq+db1hTne6QoP0vMGUv4tB0J9arQG2Ea2hHW1k3M=";
      "micro_http-0.1.0" = "sha256-gyeOop6AMXEIbLXhJMN/oYGGU8Un8Y0nFZc9ucCa0y4=";
      "mshv-bindings-0.1.1" = "sha256-vg4kStPBvHtXLuHMQzzpn4voDcVgruO+OqQ1yUCAi/U=";
      "vfio-bindings-0.4.0" = "sha256-Dk4T2dMzPZ+Aoq1YSXX2z1Nky8zvyDl7b+A8NH57Hkc=";
      "vfio_user-0.1.0" = "sha256-LJ84k9pMkSAaWkuaUd+2LnPXnNgrP5LdbPOc1Yjz5xA=";
      "vm-fdt-0.2.0" = "sha256-lKW4ZUraHomSDyxgNlD5qTaBTZqM0Fwhhh/08yhrjyE=";
      "kvm-bindings-0.7.0" = "sha256-hXv5N3TTwGQaVxdQ/DTzLt+uwLxFnstJwNhxRD2K8TM=";
      "igvm-0.1.0" = "sha256-l+Qyhdy3b8h8hPLHg5M0os8aSkjM55hAP5nqi0AGmjo=";
      "versionize_derive-0.1.6" = "sha256-eI9fM8WnEBZvskPhU67IWeN6QAPg2u5EBT+AOxfb/fY=";
    };
  };

  # Allow compilation with Rust 1.83.0, which requires public methods in
  # test modules to have documentation when the `missing_docs` lint is enabled.
  # The [Microsoft fork](https://github.com/microsoft/cloud-hypervisor) will
  # eventually support this, as [upstream](https://github.com/cloud-hypervisor/cloud-hypervisor)
  # already does.
  # See: https://github.com/cloud-hypervisor/cloud-hypervisor/issues/6903
  postPatch = ''
    substituteInPlace rate_limiter/src/lib.rs \
      --replace-fail '#![deny(missing_docs)]' ""
  '';

  patches = [
    ./0001-snp-fix-panic-when-rejecting-extended-guest-report.patch
    ./0002-hypervisor-mshv-implement-extended-guest-requests-wi.patch
  ];

  separateDebugInfo = true;

  nativeBuildInputs = [
    pkg-config
    patchelf
  ];
  buildInputs = [ openssl ];

  buildNoDefaultFeatures = true;
  buildFeatures =
    [
      "mshv"
      "kvm"
    ]
    ++ lib.optional withIGVM "igvm"
    ++ lib.optional withSEVSNP "sev_snp"
    ++ lib.optional withTDX "tdx";

  OPENSSL_NO_VENDOR = true;

  cargoTestFlags = [
    "--workspace"
    "--bins"
    "--lib" # Integration tests require root.
    "--exclude"
    "net_util" # /dev/net/tun
    "--exclude"
    "vmm" # /dev/kvm
  ];

  meta = {
    homepage = "https://github.com/microsoft/cloud-hypervisor";
    description = "Open source Virtual Machine Monitor (VMM) that runs on top of KVM";
    changelog = "https://github.com/microsoft/cloud-hypervisor/releases/tag/msft/v${version}";
    license = with lib.licenses; [
      asl20
      bsd3
    ];
    mainProgram = "cloud-hypervisor";
  };
}
