# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, rustPlatform
, fetchFromGitHub
, pkg-config
, openssl
, patchelf
, withIGVM ? true
, withSEVSNP ? true
, withTDX ? false
}:

rustPlatform.buildRustPackage rec {
  pname = "cloud-hypervisor";
  version = "32.0.317";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "cloud-hypervisor";
    rev = "refs/tags/msft/v${version}";
    hash = "sha256-D9wfCat0GVHUpppjFghKTYPl5rXE12aVxVkAFxxq78U=";
  };

  cargoLock = {
    lockFile = "${src}/Cargo.lock";
    outputHashes = {
      "acpi_tables-0.1.0" = "sha256-aT0p85QDGjBEnbABedm0q7JPpiNjhupoIzBWifQ0RaQ=";
      "micro_http-0.1.0" = "sha256-w2witqKXE60P01oQleujmHSnzMKxynUGKWyq5GEh1Ew=";
      "mshv-bindings-0.1.1" = "sha256-9Q7IXznZ+qdf/d4gO7qVEjbNUUygQDNYLNxz2BECLHc=";
      "vfio-bindings-0.4.0" = "sha256-lKdoo/bmnZTRV7RRWugwHDFFCB6FKxpzxDEEMVqSbwA=";
      "vfio_user-0.1.0" = "sha256-JYNiONQNNpLu57Pjdn2BlWOdmSf3c4/XJg/RsVxH3uk=";
      "vm-fdt-0.2.0" = "sha256-gVKGiE3ZSe+z3oOHR3zqVdc7XMHzkp4kO4p/WfK0VI8=";
      "kvm-bindings-0.6.0" = "sha256-wGdAuPwsgRIqx9dh0m+hC9A/Akz9qg9BM+p06Fi5ACM=";
      "kvm-ioctls-0.13.0" = "sha256-jHnFGwBWnAa2lRu4a5eRNy1Y26NX5MV8alJ86VR++QE=";
      "versionize_derive-0.1.4" = "sha256-BPl294UqjVl8tThuvylXUFjFNjJx8OSfBGJLg8jIkWw=";
    };
  };

  separateDebugInfo = true;

  nativeBuildInputs = [ pkg-config patchelf ];
  buildInputs = [ openssl ];

  buildNoDefaultFeatures = true;
  buildFeatures = [
    "mshv"
    "kvm"
  ] ++ lib.optional withIGVM [ "igvm" ]
  ++ lib.optional withSEVSNP [ "snp" ]
  ++ lib.optional withTDX [ "tdx" ];

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
    license = with lib.licenses; [ asl20 bsd3 ];
    mainProgram = "cloud-hypervisor";
  };
}
