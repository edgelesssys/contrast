# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  fetchurl,
  linuxManualConfig,
  stdenvNoCC,
  fetchpatch,
  kata,
  kernelconfig,
  withGPU ? false,
  withAMLSandbox ? true,
  withACPIDebug ? false,
  ... # Required for invocation through `linuxPackagesFor`, which calls this with the `features` argument.
}:
let
  configfile = stdenvNoCC.mkDerivation {
    pname = "kata-kernel-config-confidential";
    inherit (kata.release-tarball) version;

    dontUnpack = true;

    installPhase = ''
      ${lib.getExe kernelconfig} \
        ${lib.optionalString withGPU "--gpu"} \
        ${lib.optionalString withACPIDebug "CONFIG_ACPI_DEBUG=y"} \
        > $out
    '';
  };
in
linuxManualConfig rec {
  version = "6.18.15";
  modDirVersion = "${version}" + lib.optionalString withGPU "-nvidia-gpu";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    hash = "sha256-fHFiFsPEE07Q3mkZVwHmd1d7vN05efMxwYKs0Gvy8XA=";
  };

  kernelPatches = [
    # Patch prevents containers with unfixed glibc from crashing.
    # Unsure when this can be removed.
    {
      name = "work-around-the-segfault-issue-in-glibc-2-35";
      patch = fetchpatch {
        url = "https://patchwork.ozlabs.org/project/ubuntu-kernel/patch/20230123140233.790103-2-tim.gardner@canonical.com/raw/";
        hash = "sha256-kDW3yqWHxAGzaaM/5mSNoyMa2WZuyWJbMznPTPgfiyo=";
      };
    }
  ]
  ++ lib.optional withAMLSandbox {
    # This patch adds a sandbox to the AML interpreter, preventing AML code from
    # reading and writing confidential memory pages.
    # The patch is a simplified version of the original patch from Takekoshi et al.
    # as part of their "BadAML" paper, simplified by removing the Azure/Hyper-V-specific
    # parts and fittet to our use case and ported to the used kernel version.
    #
    # The paper is published here: https://dl.acm.org/doi/pdf/10.1145/3719027.3765123
    # And the original patch is part of the published research artifacts,
    # (linux/kernel/build.patch): https://zenodo.org/records/17247915
    name = "drivers-acpi-add-BadAML-sandbox";
    patch = ./0001-drivers-acpi-add-BadAML-sandbox.patch;
  };

  inherit configfile;
  allowImportFromDerivation = true;
}
