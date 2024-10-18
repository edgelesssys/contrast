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
  version = "3.10.1";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-lk9BZeNc8StLxu0frRfh7h5Xk3w75SUeZP1ddES7a20=";
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

      #
      # Patch set to enable policy support for bare metal with Nydus guest pull.
      #

      # Fixes https://github.com/kata-containers/kata-containers/issues/10064.
      # TODO(burgerdev): backport
      ./0004-genpolicy-enable-sysctl-checks.patch
      # Fixes https://github.com/kata-containers/kata-containers/issues/10065.
      # TODO(burgerdev): backport
      ./0005-genpolicy-read-bundle-id-from-rootfs.patch
      # Contrast specific layer-src-prefix, also applied to microsoft.kata-runtime.
      # TODO(burgerdev): discuss relaxing the checks for host paths with Kata maintainers.
      ./0006-genpolicy-regex-check-contrast-specific-layer-src-pr.patch
      # Kata hard-codes OCI version 1.1.0, but latest K3S has 1.2.0.
      # TODO(burgerdev): discuss relaxing the OCI version checks with Kata maintainers.
      # TODO(burgerdev): move to genpolicy-settings patches
      ./0007-genpolicy-settings-bump-OCI-version.patch
      # Nydus uses a different base dir for container rootfs,
      # see https://github.com/kata-containers/kata-containers/blob/775f6bd/tests/integration/kubernetes/tests_common.sh#L139.
      # TODO(burgerdev): discuss the discrepancy and path forward with Kata maintainers.
      ./0008-genpolicy-settings-change-cpath-for-Nydus-guest-pull.patch
      # Implements ideas from https://github.com/kata-containers/kata-containers/issues/10088.
      # TODO(burgerdev): backport
      ./0009-genpolicy-allow-image_guest_pull.patch
      # Mount configfs into the workload container from the UVM.
      # Based on https://github.com/kata-containers/kata-containers/pull/9554,
      # which wasn't accepted upstream.
      #
      # Rebase 3.8.0, changes squashed into patch:
      #   - fix 'field `annotations` of struct `oci_spec::runtime::Spec` is private'
      ./0010-runtime-agent-mounts-Mount-configfs-into-the-contain.patch
      # The following two patches update the image-rs and oci-distribution version.
      # TODO(burgerdev): backport
      ./0011-genpolicy-bump-oci-distribution-to-v0.12.0.patch

      # This is an alternative implementation of
      # packages/by-name/microsoft/genpolicy/0005-genpolicy-propagate-mount_options-for-empty-dirs.patch
      # that does not depend on the CSI enabling changes exclusive to the Microsoft fork.
      ./0012-genpolicy-support-mount-propagation-and-ro-mounts.patch
      # Prevent cleanup of the build root to allow adding files before running rootfs.sh.
      # This allows working around a bug in the script, which assumes existence of a file that's
      # only added later:
      # https://github.com/kata-containers/kata-containers/blame/94bc54f4d21fe74e078880a6b5f9f96137a9e6bb/tools/osbuilder/rootfs-builder/rootfs.sh#L723.
      # The patch is not sufficient for upstream, because it requires the extraRootFs content from
      # our Nix packaging.
      ./0013-tools-don-t-clean-build-root-when-generating-rootfs.patch

      # Fixes https://github.com/kata-containers/kata-containers/issues/10424
      # Those patches only got merged after v3.10.1 was released
      # Drop when upgrading to v3.11+
      ./0014-kata-sys-util-remove-obsolete-cgroups-dependency.patch
      ./0015-kata-sys-util-move-json-parsing-to-protocols-crate.patch
      ./0016-protocols-only-build-RLimit-impls-on-Linux.patch

      # A peer-pod VM does not have HOSTDATA or MRCONFIGID, so the expected policy hash needs to
      # be configured differently. This patch adds a policy hash config field to the agent config,
      # which is passed by the CAA and loaded from user-data.
      # The upstream plan-of-record is the initdata proposal, which will eventually provide all
      # podvm configuration in a measurable way. Unfortunately, this proposal has diverged between
      # Kata and CAA, so we're implementing our own solution here.
      ./0017-agent-read-policy-hash-from-config.patch
      # This patch makes the remote hypervisor aware of the workload policy.
      ./0018-runtime-forward-policy-to-remote-hypervisor.patch
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
