# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  buildGoModule,
  fetchFromGitHub,
  yq-go,
  git,
  applyPatches,
}:

buildGoModule (finalAttrs: {
  pname = "kata-runtime";
  version = "3.17.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = finalAttrs.version;
      hash = "sha256-rYF9YIZ8GdiE12QfX4rDXVPb7umuIhsLXoWmRl3oesk=";
    };

    patches = [
      ./0001-emulate-CPU-model-that-most-closely-matches-the-host.patch
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
      ./0002-runtime-agent-verify-the-agent-policy-hash.patch

      #
      # Patch set to enable policy support for bare metal with Nydus guest pull.
      #

      # Fixes https://github.com/kata-containers/kata-containers/issues/10065.
      # TODO(burgerdev): backport
      ./0003-genpolicy-read-bundle-id-from-rootfs.patch
      # Contrast specific layer-src-prefix, also applied to microsoft.kata-runtime.
      # TODO(burgerdev): discuss relaxing the checks for host paths with Kata maintainers.
      ./0004-genpolicy-regex-check-contrast-specific-layer-src-pr.patch
      # An attacker can set any OCI version they like, so we can't rely on it.
      # The policy must be secure no matter what OCI version is communicated.
      # TODO(kateoxchen): upstream. See https://github.com/kata-containers/kata-containers/issues/10632.
      # TODO(katexochen): Additional security measures should be taken to ensure the OCI
      # version is the same well use to create the container and the policy covers all the
      # fields of the spec.
      ./0005-genpolicy-rules-remove-check-for-OCI-version.patch
      # Nydus uses a different base dir for container rootfs,
      # see https://github.com/kata-containers/kata-containers/blob/775f6bd/tests/integration/kubernetes/tests_common.sh#L139.
      # TODO(burgerdev): discuss the discrepancy and path forward with Kata maintainers.
      ./0006-genpolicy-settings-change-cpath-for-Nydus-guest-pull.patch
      # Implements ideas from https://github.com/kata-containers/kata-containers/issues/10088.
      # TODO(burgerdev): backport
      ./0007-genpolicy-allow-image_guest_pull.patch
      # Mount configfs into the workload container from the UVM.
      # Based on https://github.com/kata-containers/kata-containers/pull/9554,
      # which wasn't accepted upstream.
      #
      # Rebase 3.8.0, changes squashed into patch:
      #   - fix 'field `annotations` of struct `oci_spec::runtime::Spec` is private'
      ./0008-runtime-agent-mounts-Mount-configfs-into-the-contain.patch

      # This is an alternative implementation of
      # packages/by-name/microsoft/genpolicy/0005-genpolicy-propagate-mount_options-for-empty-dirs.patch
      # that does not depend on the CSI enabling changes exclusive to the Microsoft fork.
      ./0009-genpolicy-support-mount-propagation-and-ro-mounts.patch

      # Disable a check in Kata that prevents to set both image and initrd.
      # For us, there's no practical reason not to do so.
      # No upstream patch available, changes first need to be discussed with Kata maintainers.
      # See https://katacontainers.slack.com/archives/C879ACQ00/p1731928491942299
      ./0010-runtime-allow-initrd-AND-image-to-be-set.patch

      # Simple genpolicy logging redaction of the policy annotation
      # This avoids printing the entire annotation on log level debug, which resulted in errors of the logtranslator.go
      # upstream didn't accept this patch: https://github.com/kata-containers/kata-containers/pull/10647
      ./0011-genpolicy-do-not-log-policy-annotation-in-debug.patch

      # Fixes a bug with ConfigMaps exceeding 8 entries, see description.
      # The situation upstream is complicated, because the paths relevant for genpolicy differ
      # between different CI systems and TEE configurations. This makes it hard to reproduce in a
      # vanilla Kata setting.
      # Relevant discussion: https://github.com/kata-containers/kata-containers/pull/10614.
      ./0012-genpolicy-allow-non-watchable-ConfigMaps.patch

      # Guest hooks are required for GPU support, but unsupported in
      # upstream Kata / genpolicy as of now. This patch adds a new
      # `allowed_guest_hooks` setting , which controls what paths may be set for hooks.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/10633
      ./0013-genpolicy-support-guest-hooks.patch

      # This adds support for annotations with dynamic keys *and* values to Genpolicy.
      # This is required for e.g. GPU containers, which get annotated by an in-cluster
      # component (i.e. after policy generation based on the Pod spec) with an annotation
      # like `cdi.k8s.io/vfioXY`, where `XY` corresponds to a dynamic ID.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/10745
      ./0014-genpolicy-support-dynamic-annotations.patch

      # This removes CDI annotations from the OCI spec before it is passed to the agent,
      # which helps with policy handling of the (oftentimes dynamic) CDI annotations.
      # TODO(msanft): Get native CDI working, which will allow us to drop this patch / undo the revert.
      # See https://dev.azure.com/Edgeless/Edgeless/_workitems/edit/5061
      ./0015-runtime-remove-CDI-annotations.patch

      # Allow running generate with ephemeral volumes.
      #
      # This may be merged upstream through either of:
      # - https://github.com/kata-containers/kata-containers/pull/10947 (this patch)
      # - https://github.com/kata-containers/kata-containers/pull/10559 (superset including the patch)
      ./0016-genpolicy-support-ephemeral-volume-source.patch

      # Containerd versions since 2.0.4 set the sysfs of the pause container to RW if one of the
      # main containers is privileged, whereas prior versions did not. The expected mounts are
      # hard-coded in containerd.rs, making it tricky to support differences across containerd
      # versions. We deal with this by always configuring the pause container's sysfs as RW, and
      # then allowing a mount that is expected to be RW to be mounted as RO. The worst thing that
      # could happen here is a container failure during sysfs writes.
      #
      # This workaround would not be necessary if we had better support for diverse containerd
      # versions upstream. However, there is no consensus on how this would look like, or whether
      # it makes sense at all, so we're fixing this downstream only.
      # https://github.com/kata-containers/kata-containers/pull/11077#issuecomment-2750400613
      ./0017-genpolicy-allow-RO-and-RW-for-sysfs-with-privileged-.patch

      # Support to enforce guest pull without (nydus) snapshotter.
      # Cherry-picked from https://github.com/kata-containers/kata-containers/pull/11244
      ./0018-runtime-add-option-to-force-guest-pull.patch

      # Fixes a bug in the genpolicy settings where the service_name regex used to match downward
      # API env vars wouldn't accept numbers in the service name.
      # Upstream PR: https://github.com/kata-containers/kata-containers/pull/11314
      ./0019-genpolicy-fix-svc_name-regex.patch
      ./0020-genpolicy-rename-svc_name-to-svc_name_downward_env.patch

      # Exec requests are failing on Kata, as allow_interactive_exec is blocking execution.
      # Reason for this is that a subsequent check asserts the sandbox-name from the annotations, but such annotation
      # is only added for pods by genpolicy. The sandbox name of other pod-generating resources is hard to predict.
      #
      # With this patch, we use a regex check for the sandbox name in these cases. We construct the regex in genpolicy
      # based on the the specified metadata, following the logic after which kubernetes will derive the sandbox name.
      # The generated regex is then used in the policy to match the sandbox name.
      #
      # TODO(burgerdev): upstream
      ./0021-genpolicy-match-sandbox-name-by-regex.patch

      # The following two patches remove irrelevant warnings from genpolicy that clutter our CLI output.
      # Upstream PR: https://github.com/kata-containers/kata-containers/pull/11358.
      ./0022-genpolicy-remove-redundant-group-check.patch
      ./0023-genpolicy-push-down-warning-about-missing-passwd-fil.patch
    ];
  };

  sourceRoot = "${finalAttrs.src.name}/src/runtime";

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
})
