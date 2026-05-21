# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Shared source meta-package for the kata-containers fork.
#
# Owns:
#   - the upstream version pin and `fetchFromGitHub` (`srcRaw`)
#   - the patch set and the patched source (`src`)
#   - the `cargoLock.outputHashes` map for all git deps in the kata
#     workspace (`outputHashes`)
#   - a single shared vendor dir for the workspace's `Cargo.lock`
#     (`cargoVendorDir`), wrapped so crane's setup hook accepts it.
#
# Consumed by `kata.runtime`, `kata.agent`, `kata.runtime-rs`, and
# `kata.genpolicy` via the nested by-name scope (each receives this as
# its `source` argument).

{
  fetchFromGitHub,
  applyPatches,
  rustPlatform,
  runCommand,
  lib,
  craneLib,
}:

rec {
  version = "3.31.0";

  srcRaw = fetchFromGitHub {
    owner = "kata-containers";
    repo = "kata-containers";
    rev = version;
    hash = "sha256-LSwmeNO5mEWZ2NCrC1o3JXPeZJed0eljIUak/J2fkv0=";
  };

  src = applyPatches {
    src = srcRaw;

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

      # Use virtio-blk with serial name for initdata
      # Our initdata-processor expects the initdata device to be present at /dev/disks/by-label/initdata,
      # which requires the device to have a stable name. Using virtio-blk with a serial number achieves this.
      # TODO: check if we can improve the situation upstream or implement a fallback in the initdata-processor.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/12764.
      ./0022-runtime-rs-force-virtio-blk-with-serial-name-for-ini.patch

      # We pass a custom config to the Kata agent via commandline argument to enable the debug console and customize logging.
      # The provided config should only be used as an override and not as a replacement, so the kernel cmdline is still respected
      # when we don't have any overrides.
      ./0023-agent-use-config-file-as-override.patch

      # Terminate the agent gracefully when it encounters problems with the policy. This allows
      # error messages to propagate to the runtime.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/13031.
      ./0024-agent-don-t-abort-in-case-of-policy-problems.patch

      # Pod-level resource limits are a beta feature since K8s 1.34, but not supported by Kata yet.
      # We need this to automatically configure memory limits for the entire VM and not on container basis.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/12816
      ./0025-genpolicy-support-pod-level-resource-limits.patch

      # We need the layer sizes to compute the overhead of pulling the images into the VM.
      # This caches the compressed and uncompressed layer sizes in the layers-cache.json file during genpolicy.
      # The layers-cache.json file is no longer indexed by diffID, but by the layer digest,
      # which is the only stable identifier for the compressed layer size.
      ./0026-genpolicy-cache-un-compressed-layer-sizes.patch

      # This writes a subset of the layers-cache.json file into a separate file containing only the processed layers.
      # We need the layer information to calculate the memory overhead for the VM during generate.
      ./0027-genpolicy-write-processed-layer-information-to-file.patch

      # Stop the kata shim's cleanup paths from hanging when the kata-agent
      # is unreachable. Without this, agent calls hang in commonDialer past
      # containerd's 5s per-shim cleanup deadline, the task dir is orphaned,
      # and accumulated leaks cascade through kubelet's Requires=containerd.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/11328.
      # TODO(sse): retire this carry once contrast migrates to runtime-rs.
      ./0028-runtime-stop-shim-cleanup-from-hanging-on-a-dead-kat.patch
    ];
  };

  outputHashes = {
    "api_client-0.1.0" = "sha256-RdwQg6/EI+oGkyNXnu5t1q87oTXev25XpIaE+PWDTx4=";
    "cgroups-rs-0.3.5" = "sha256-BKD1ZPK5LqB/n2xD/oODArVKjbH+MQOeYn/UYbBHzn0=";
    "micro_http-0.1.0" = "sha256-XemdzwS25yKWEXJcRX2l6QzD7lrtroMeJNOUEWGR7WQ=";
    "regorus-0.9.1" = "sha256-+TCq9r8kTNM0URbcDP4D9/lKA6Bni7+KgrGRTJFbQPM=";
    "s390_pv_core-0.11.0" = "sha256-P275gUoF4JtaKvKPvzhCsBuo882kKCYebtNpCDEmTP0=";
  };

  # Vendor dir keyed on the *unpatched* workspace Cargo.lock so that adding
  # or removing non-Cargo patches doesn't invalidate the cache for crane
  # consumers.
  cargoVendorDir = runCommand "kata-cargo-vendor-${version}" { } ''
    mkdir -p $out
    cp -r --no-preserve=mode,ownership ${
      rustPlatform.importCargoLock {
        lockFile = "${srcRaw}/Cargo.lock";
        inherit outputHashes;
      }
    }/. $out/
    substituteInPlace $out/.cargo/config.toml \
      --replace-fail 'directory = "cargo-vendor-dir"' "directory = \"$out\""
    cp $out/.cargo/config.toml $out/config.toml
  '';

  # Builds a `cargoArtifacts` chain that pre-compiles the kata workspace with the given source dir filtered out.
  # This allows edits in that dir to not affect the cache status of the other packages in the workspace.
  #
  # Captures `src/libs/protocols/src/` because the protocols build script generates code into the source tree.
  # Consumers must restore it via `restoreProtocolsSrc` before their cargo invocation!
  mkCargoArtifacts =
    {
      pname,
      cargoExtraArgs,
      nativeBuildInputs,
      buildInputs,
      env,
      strictDeps ? true,
      stubPrefix,
      stubScript,
      preBuild ? "chmod -R +w .\n",
    }:
    craneLib.cargoBuild {
      inherit
        pname
        version
        cargoVendorDir
        strictDeps
        cargoExtraArgs
        nativeBuildInputs
        buildInputs
        env
        preBuild
        ;
      pnameSuffix = "-workspace";
      doCheck = false;

      src = runCommand "source-patched" { } ''
        cp -r --no-preserve=mode,ownership ${
          lib.cleanSourceWith {
            name = "source-patched-stubbed";
            inherit src;
            filter = path: _type: !(lib.hasInfix "/${stubPrefix}/" path);
          }
        }/. $out/
        chmod -R +w $out
        mkdir -p $out/${stubPrefix}
        ${stubScript}
      '';

      postInstall = ''
        mkdir -p $out/source-mods
        cp -r src/libs/protocols/src $out/source-mods/protocols-src
      '';

      cargoArtifacts = craneLib.buildDepsOnly {
        inherit
          pname
          version
          cargoVendorDir
          strictDeps
          cargoExtraArgs
          nativeBuildInputs
          buildInputs
          env
          preBuild
          ;
        src = srcRaw;
      };
    };

  restoreProtocolsSrc = ''
    cp -af "$cargoArtifacts/source-mods/protocols-src/." src/libs/protocols/src/
    chmod -R +w src/libs/protocols/src/
  '';
}
