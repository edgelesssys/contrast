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
  version = "3.13.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-xBEK+Tczc4MVnETx5sV9sb5/myxLeP7YDDigTroN4Lg=";
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
      # An attacker can set any OCI version they like, so we can't rely on it.
      # The policy must be secure no matter what OCI version is communicated.
      # TODO(kateoxchen): upstream. See https://github.com/kata-containers/kata-containers/issues/10632.
      # TODO(katexochen): Additional security measures should be taken to ensure the OCI
      # version is the same well use to create the container and the policy covers all the
      # fields of the spec.
      ./0007-genpolicy-rules-remove-check-for-OCI-version.patch
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

      # Disable a check in Kata that prevents to set both image and initrd.
      # For us, there's no practical reason not to do so.
      # No upstream patch available, changes first need to be discussed with Kata maintainers.
      # See https://katacontainers.slack.com/archives/C879ACQ00/p1731928491942299
      ./0014-runtime-allow-initrd-AND-image-to-be-set.patch

      # Simple genpolicy logging redaction of the policy annotation
      # This avoids printing the entire annotation on log level debug, which resulted in errors of the logtranslator.go
      # TODO(jmxnzo): remove when upstream patch is merged: https://github.com/kata-containers/kata-containers/pull/10647
      ./0015-genpolicy-do-not-log-policy-annotation-in-debug.patch

      # Fixes a bug with ConfigMaps exceeding 8 entries, see description.
      # The situation upstream is complicated, because the paths relevant for genpolicy differ
      # between different CI systems and TEE configurations. This makes it hard to reproduce in a
      # vanilla Kata setting.
      # Relevant discussion: https://github.com/kata-containers/kata-containers/pull/10614.
      ./0016-genpolicy-allow-non-watchable-ConfigMaps.patch

      # Guest hooks are required for GPU support, but unsupported in
      # upstream Kata / genpolicy as of now. This patch adds a new
      # `allowed_guest_hooks` setting , which controls what paths may be set for hooks.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/10633
      ./0017-genpolicy-support-guest-hooks.patch

      # Revert CDI support in kata-agent, which breaks legacy mode GPU facilitation which
      # we currently use.
      # TODO(msanft): Get native CDI working, which will allow us to drop this patch / undo the revert.
      # See https://dev.azure.com/Edgeless/Edgeless/_workitems/edit/5061
      ./0018-agent-remove-CDI-support.patch

      # This adds support for annotations with dynamic keys *and* values to Genpolicy.
      # This is required for e.g. GPU containers, which get annotated by an in-cluster
      # component (i.e. after policy generation based on the Pod spec) with an annotation
      # like `cdi.k8s.io/vfioXY`, where `XY` corresponds to a dynamic ID.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/10745
      ./0019-genpolicy-support-dynamic-annotations.patch

      # This allows denying ReadStream requests without blocking the container on its
      # stdout/stderr, by redacting the streams instead of blocking them.
      # Upstream:
      # * https://github.com/kata-containers/kata-containers/issues/10680
      # * https://github.com/kata-containers/kata-containers/pull/10818
      ./0020-agent-clear-log-pipes-if-denied-by-policy.patch
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

  # Default commandline used by Kata containers.
  # This has no single source upstream, but can be derived by manually checking what command-line
  # is used when Kata starts a VM.
  # For example, this command should do the job:
  # `journalctl -t kata -l --no-pager | grep launching | tail -1`
  passthru.cmdline = {
    default = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none";
    debug = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 debug systemd.show_status=true systemd.log_level=debug panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026";
  };

  meta.mainProgram = "containerd-shim-kata-v2";
}
