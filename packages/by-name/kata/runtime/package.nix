# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  fetchFromGitHub,
  yq-go,
  git,
  applyPatches,
}:

buildGoModule (finalAttrs: {
  pname = "kata-runtime";
  version = "3.28.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = finalAttrs.version;
      hash = "sha256-gf1z2lHRN9wRFTN6l62pBE1aeUXOv1auMGhloKRY6WQ=";
    };

    patches = [
      ./0001-emulate-CPU-model-that-most-closely-matches-the-host.patch

      #
      # Patch set to enable policy support for bare metal with Nydus guest pull.
      #

      # An attacker can set any OCI version they like, so we can't rely on it.
      # The policy must be secure no matter what OCI version is communicated.
      # TODO(kateoxchen): upstream. See https://github.com/kata-containers/kata-containers/issues/10632.
      # TODO(katexochen): Additional security measures should be taken to ensure the OCI
      # version is the same well use to create the container and the policy covers all the
      # fields of the spec.
      ./0002-genpolicy-rules-remove-check-for-OCI-version.patch
      # Implements ideas from https://github.com/kata-containers/kata-containers/issues/10088.
      # TODO(burgerdev): backport
      ./0003-genpolicy-allow-image_guest_pull.patch
      # Mount configfs into the workload container from the UVM.
      # Based on https://github.com/kata-containers/kata-containers/pull/9554,
      # which wasn't accepted upstream.
      #
      # Rebase 3.8.0, changes squashed into patch:
      #   - fix 'field `annotations` of struct `oci_spec::runtime::Spec` is private'
      ./0004-runtime-agent-mounts-Mount-configfs-into-the-contain.patch

      # This is an alternative implementation of
      # packages/by-name/microsoft/genpolicy/0005-genpolicy-propagate-mount_options-for-empty-dirs.patch
      # that does not depend on the CSI enabling changes exclusive to the Microsoft fork.
      ./0005-genpolicy-support-mount-propagation-and-ro-mounts.patch

      # Disable a check in Kata that prevents to set both image and initrd.
      # For us, there's no practical reason not to do so.
      # No upstream patch available, changes first need to be discussed with Kata maintainers.
      # See https://katacontainers.slack.com/archives/C879ACQ00/p1731928491942299
      ./0006-runtime-allow-initrd-AND-image-to-be-set.patch

      # Simple genpolicy logging redaction of the policy annotation
      # This avoids printing the entire annotation on log level debug, which resulted in errors of the logtranslator.go
      # upstream didn't accept this patch: https://github.com/kata-containers/kata-containers/pull/10647
      ./0007-genpolicy-do-not-log-policy-annotation-in-debug.patch

      # Allow running generate with ephemeral volumes.
      #
      # This may be merged upstream through either of:
      # - https://github.com/kata-containers/kata-containers/pull/10947 (this patch)
      # - https://github.com/kata-containers/kata-containers/pull/10559 (superset including the patch)
      ./0008-genpolicy-support-ephemeral-volume-source.patch

      # Don't add storages for volumes declared in the image config.
      # This fixes a security issue where the host is able to write untrusted content to paths
      # under these volumes, by failing the policy generation if volumes without mounts are found.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/11546.
      ./0009-genpolicy-don-t-allow-mount-storage-for-declared-VOL.patch

      # Imagepulling has moved into the CDH in Kata 3.18.0. Since we are not using the CDH,we are instead starting our own Imagepuller.
      # This patch redirects calls by upstream's PullImage ttRPC client implementation to communicate with our imagepuller ttRPC server.
      # The patch should become unnecessary once the RFC for loose coupling of agents and guest components is implemented:
      # https://github.com/kata-containers/kata-containers/issues/11532
      ./0010-agent-use-custom-implementation-for-image-pulling.patch

      # Changes the unix socket used for ttRPC communication with the imagepuller.
      # Necessary to allow a separate imagestore service.
      # Can be removed in conjunction with patch 0018-agent-use-custom-implementation-for-image-pulling.patch.
      ./0011-agent-use-separate-unix-socket-for-image-pulling.patch

      # Secure mounting is part of the CDH in Kata. Since we are not using the CDH, we are instead reimplementing it.
      # This patch redirects calls by upstream's SecureImageStore ttRPC client implementation to communicate with our own ttRPC server.
      # The patch should become unnecessary once the RFC for loose coupling of agents and guest components is implemented:
      # https://github.com/kata-containers/kata-containers/issues/11532
      ./0012-agent-use-custom-implementation-for-secure-mounting.patch

      # Upstream expects guest pull to only use Nydus and applies workarounds that are not
      # necessary with force_guest_pull. This patch removes the workaround.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/11757.
      ./0013-genpolicy-don-t-apply-Nydus-workaround.patch

      # We're using a dedicated initdata-processor job and don't want the Kata agent to manage
      # initdata for us.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/11532.
      ./0014-agent-remove-initdata-processing.patch

      # In addition to the initdata device, we also require the imagepuller's auth config
      # to be passed to the VM in a similar manner.
      ./0015-runtime-pass-imagepuller-config-device-to-vm.patch

      # Privatemode requires GPU sharing between containers of the same pod.
      # In the hook-based flow, this worked because all devices and libs were (accidentally) handed to all containers.
      # With the CDI-based flow, this no longer happens.
      # Instead, this patch ensures that if a container has NVIDIA_VISIBLE_DEVICES=all set as an env var,
      # that container receives ALL Nvidia GPU devices known to the pod.
      ./0016-runtime-assign-GPU-devices-to-multiple-containers.patch

      # With recent versions of the sandbox-device-plugin, a /dev/iommu device is added
      # to the container spec for GPU-enabled containers.
      # Since the same thing is done by the CTK within the PodVM, and we only want this
      # to influence VM creation, we remove this device from the container spec in the agent.
      # Upstream bug: https://github.com/kata-containers/kata-containers/issues/12246.
      ./0017-runtime-remove-iommu-device.patch

      # We are observing frequent pull failures from genpolicy due to the connection being reset by the registry.
      # This patch allows genpolicy to retry these failed pulls multiple times.
      # Upstream PR: https://github.com/kata-containers/kata-containers/pull/12300.
      ./0018-genpolicy-retry-failed-image-pulls.patch

      # In clusters that don't use the sandbox-device-plugin's P_GPU_ALIAS, we will not be able to
      # look up the device via PodResources. This patch adds additional resolution logic for that
      # case, relaxing the matching requirement to just the name (without vendor and class).
      # This is unlikely to be fixed in Kata upstream, but rather in the NVIDIA components.
      # Upstream issue: https://github.com/NVIDIA/sandbox-device-plugin/issues/46
      ./0019-shim-guess-CDI-devices-without-direct-match.patch

      # Kata takes a default_maxvcpus config option. Ordinarily, we could set this to 240 and do the same in the kernel commandline below.
      # However, kata then reduces this number to the actually available number of CPUs at runtime.
      # This is a problem for us because we need to know the precise kernel command line at buildtime.
      # TODO(charludo): attempt to make this behavior configurable upstream
      ./0020-runtime-do-not-add-nr_vcpus-to-kernel-command-line.patch

      # Enables the Kata runtime to set the SNP ID blocks for the CPU model it is running on
      # based on Pod annotations. This allows us to run Pods with multiple CPUs.
      # This patch relies on changes made by 0001-emulate-CPU-model-that-most-closely-matches-the-host.patch
      # together with being specific to our use case. There are no plans to upstream it.
      ./0021-runtime-add-SNP-ID-block-from-Pod-annotations.patch

      # Fixes a vulnerability in the CopyFile policy.
      # Drop after upgrading to Kata 3.29.0.
      # https://github.com/kata-containers/kata-containers/security/advisories/GHSA-q49m-57vm-c8cc
      ./0022-genpolicy-restrict-symlinks-in-CopyFile.patch
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
  #
  # Notice! Ordering matters and depends on the order in which kata-runtime adds these values.
  passthru.cmdline =
    let
      cmdline =
        debug:
        lib.concatStringsSep " " (
          [
            "tsc=reliable"
            "no_timer_check"
            "rcupdate.rcu_expedited=1"
            "i8042.direct=1"
            "i8042.dumbkbd=1"
            "i8042.nopnp=1"
            "i8042.noaux=1"
            "noreplace-smp"
            "reboot=k"
            "cryptomgr.notests"
            "net.ifnames=0"
            "pci=lastbus=0"
            "root=/dev/vda1"
            "rootflags=ro"
            "rootfstype=erofs"
            "console=hvc0"
            "console=hvc1"
          ]
          ++ lib.optionals debug [
            "debug"
            "systemd.show_status=true"
            "systemd.log_level=debug"
          ]
          ++ lib.optionals (!debug) [
            "quiet"
            "systemd.show_status=false"
          ]
          ++ [
            "panic=1"
            "selinux=0"
            "systemd.unit=kata-containers.target"
            "systemd.mask=systemd-networkd.service"
            "systemd.mask=systemd-networkd.socket"
            "scsi_mod.scan=none"
          ]
          ++ lib.optionals debug [
            "agent.log=debug"
            "agent.debug_console"
            "agent.debug_console_vport=1026"
          ]
        );
    in
    {
      default = cmdline false;
      debug = cmdline true;
    };

  meta.mainProgram = "containerd-shim-kata-v2";
})
