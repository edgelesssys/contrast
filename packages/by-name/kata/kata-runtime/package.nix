# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildGoModule,
  fetchFromGitHub,
  yq-go,
  git,
  applyPatches,
}:

buildGoModule rec {
  pname = "kata-runtime";
  version = "3.8.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-62qoAMlE62hS02+Bj5HNgNyGVTk7SVLJaqN9GhCWQXc=";
    };

    patches = [
      ./0001-govmm-Directly-pass-the-firwmare-using-bios-with-SNP.patch
      ./0002-emulate-CPU-model-that-most-closely-matches-the-host.patch
      # This patch makes the v2 shim set the host-data field for SNP and makes
      # kata-agent verify it against the policy.
      # It was adapted from https://github.com/kata-containers/kata-containers/pull/8469,
      # with the following modifications:
      # - Rebase on 3.7.0 picked up regorus, guest-pull capabilities, SNP certificates and TDX fixes.
      # - The TDX parameters for QEMU needed to be converted to a JSON object.
      # - The encoding of MRCONFIGID needed to be switched from hex to base64.
      # - Don't allow skipping the policy checks.
      # - Always use sha256 - even for TDX.
      # This patch is not going to be accepted upstream. The declared path
      # forward is the initdata proposal, https://github.com/kata-containers/kata-containers/issues/9468,
      # which extends the hostdata to arbitrary config beyond the policy and
      # delegates hash verification to the AA. Until that effort lands, we're
      # sticking with the policy verification from AKS CoCo.
      ./0003-runtime-agent-verify-the-agent-policy-hash.patch
      ./0004-virtcontainers-allow-specifying-nydus-overlayfs-bina.patch

      #
      # Patch set to enable policy support for bare metal with Nydus guest pull.
      #

      # Backport of https://github.com/kata-containers/kata-containers/pull/9911.
      # TODO(burgerdev): remove after upgrading to Kata 3.9
      ./0005-genpolicy-deny-UpdateEphemeralMountsRequest.patch
      # Cherry-pick from https://github.com/microsoft/kata-containers/pull/139/commits/e4465090e693807d6ccc044344ad44789acda3e2,
      # fixes https://github.com/kata-containers/kata-containers/issues/10046.
      # Currently not possible to backport because it would break integration testing with virtiofs.
      ./0006-genpolicy-validate-create-sandbox-storages.patch
      # Fixes https://github.com/kata-containers/kata-containers/issues/10064.
      # TODO(burgerdev): backport
      ./0007-genpolicy-enable-sysctl-checks.patch
      # Fixes https://github.com/kata-containers/kata-containers/issues/10065.
      # TODO(burgerdev): backport
      ./0008-genpolicy-read-bundle-id-from-rootfs.patch
      # Contrast specific layer-src-prefix, also applied to microsoft.kata-runtime.
      # TODO(burgerdev): discuss relaxing the checks for host paths with Kata maintainers.
      ./0009-genpolicy-regex-check-contrast-specific-layer-src-pr.patch
      # Kata hard-codes OCI version 1.1.0, but latest K3S has 1.2.0.
      # TODO(burgerdev): discuss relaxing the OCI version checks with Kata maintainers.
      # TODO(burgerdev): move to genpolicy-settings patches
      ./0010-genpolicy-settings-bump-OCI-version.patch
      # Nydus uses a different base dir for container rootfs,
      # see https://github.com/kata-containers/kata-containers/blob/775f6bd/tests/integration/kubernetes/tests_common.sh#L139.
      # TODO(burgerdev): discuss the discrepancy and path forward with Kata maintainers.
      ./0011-genpolicy-settings-change-cpath-for-Nydus-guest-pull.patch
      # Implements ideas from https://github.com/kata-containers/kata-containers/issues/10088.
      # TODO(burgerdev): backport
      ./0012-genpolicy-allow-image_guest_pull.patch
    ];
  };

  sourceRoot = "${src.name}/src/runtime";

  vendorHash = null;

  subPackages = [
    "cmd/containerd-shim-kata-v2"
    "cmd/kata-monitor"
    "cmd/kata-runtime"
  ];

  preBuild = ''
    substituteInPlace Makefile \
      --replace-fail 'include golang.mk' ""
    for f in $(find . -name '*.in' -type f); do
      make ''${f%.in}
    done
  '';

  nativeBuildInputs = [
    yq-go
    git
  ];

  ldflags = [ "-s" ];

  # Hack to skip all kata-runtime tests, which require a Git repo.
  preCheck = ''
    rm cmd/kata-runtime/*_test.go
  '';

  checkFlags =
    let
      # Skip tests that require a working hypervisor
      skippedTests = [
        "TestArchRequiredKernelModules"
        "TestCheckCLIFunctionFail"
        "TestEnvCLIFunction(|Fail)"
        "TestEnvGetAgentInfo"
        "TestEnvGetEnvInfo(|SetsCPUType|NoHypervisorVersion|AgentError|NoOSRelease|NoProcCPUInfo|NoProcVersion)"
        "TestEnvGetRuntimeInfo"
        "TestEnvHandleSettings(|InvalidParams)"
        "TestGetHypervisorInfo"
        "TestGetHypervisorInfoSocket"
        "TestSetCPUtype"
      ];
    in
    [ "-skip=^${builtins.concatStringsSep "$|^" skippedTests}$" ];

  meta.mainProgram = "containerd-shim-kata-v2";
}
