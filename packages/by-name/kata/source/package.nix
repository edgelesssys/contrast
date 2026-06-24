# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  fetchFromGitHub,
  applyPatches,
  rustPlatform,
  callPackage,
  protobuf,
  pkg-config,
  openssl,
  libseccomp,
  lvm2,
  cmake,
  zlib,
}:

rec {
  version = "3.32.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-dnbzjYDKeAp0wFQcO5VK71vkf7ubVK5Lh9R9jjuro28=";
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

      # Enables the Kata runtime to set the SNP ID blocks for the CPU model it is running on
      # based on Pod annotations. This allows us to run Pods with multiple CPUs.
      # This patch relies on changes made by 0001-emulate-CPU-model-that-most-closely-matches-the-host.patch
      # together with being specific to our use case. There are no plans to upstream it.
      ./0020-runtime-add-SNP-ID-block-from-Pod-annotations.patch

      # We pass a custom config to the Kata agent via commandline argument to enable the debug console and customize logging.
      # The provided config should only be used as an override and not as a replacement, so the kernel cmdline is still respected
      # when we don't have any overrides.
      ./0021-agent-use-config-file-as-override.patch

      # Terminate the agent gracefully when it encounters problems with the policy. This allows
      # error messages to propagate to the runtime.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/13031.
      ./0022-agent-don-t-abort-in-case-of-policy-problems.patch

      # Stop the kata shim's cleanup paths from hanging when the kata-agent
      # is unreachable. Without this, agent calls hang in commonDialer past
      # containerd's 5s per-shim cleanup deadline, the task dir is orphaned,
      # and accumulated leaks cascade through kubelet's Requires=containerd.
      # Upstream issue: https://github.com/kata-containers/kata-containers/issues/11328.
      # TODO(sse): retire this carry once contrast migrates to runtime-rs.
      ./0023-runtime-stop-shim-cleanup-from-hanging-on-a-dead-kat.patch

      # Don't clean up the Kata cgroup when cleaning up a failed pod. It's unnecessary because it
      # will be cleaned up by containerd, and it may hide problems if the cleanup takes too long.
      # No upstream issue because this is unlikely to be accepted upstream as is, and developing a
      # full fix for all potential configurations is not a good investment of time.
      # TODO(burgerdev): drop patch after migrating to runtime-rs
      ./0024-runtime-don-t-attempt-to-clean-up-cgroup-scope.patch

      # Cache (un-)compressed image sizes and information to map image references to DiffIDs in the layers-cache.json.
      # For each image reference, compressed sizes are taken from the image manifest and are stored with a reference
      # to the corresponding DiffID. Entries with a DiffID key now also store the uncompressed size of the layer.
      # This allows us to calculate the pod memory requirements based on the image sizes.
      ./0025-genpolicy-cache-image-sizes-and-layer-info.patch
    ];
  };

  cargoNixPackage = callPackage ./Cargo.nix {
    workspaceSrc = src;
    buildRustCrateForPkgs =
      pkgs: crate:
      (pkgs.buildRustCrate.override {
        inherit (pkgs.buildPackages) rustc cargo;
        defaultCodegenUnits = 16;
        defaultCrateOverrides = pkgs.defaultCrateOverrides // {
          openssl-sys = _: {
            OPENSSL_NO_VENDOR = 1;
          };
          # libseccomp and lvm2 are Linux-only, so they can't go in the shared buildInputs above
          libseccomp-sys = _: {
            buildInputs = [
              libseccomp
              libseccomp.dev
              libseccomp.lib
            ];
          };
          kata-agent = _: {
            buildInputs = [
              rustPlatform.bindgenHook
              lvm2.dev
              libseccomp
              libseccomp.dev
              libseccomp.lib
            ];
            LIBC = "gnu";
            preBuild = ''
              substitute src/version.rs.in src/version.rs \
                --replace-fail @AGENT_VERSION@ ${version} \
                --replace-fail @API_VERSION@ 0.0.1 \
                --replace-fail @VERSION_COMMIT@ ${version} \
                --replace-fail @COMMIT@ ""
            '';
          };
          shim = _: {
            preBuild = ''
              substitute src/config.rs.in src/config.rs \
                --replace-fail @PROJECT_NAME@ "Kata Containers" \
                --replace-fail @RUNTIME_VERSION@ ${version} \
                --replace-fail @COMMIT@ none \
                --replace-fail @RUNTIME_NAME@ containerd-shim-kata-v2 \
                --replace-fail @CONTAINERD_RUNTIME_NAME@ io.containerd.kata.v2
            '';
          };
          genpolicy = _: {
            preBuild = ''
              substitute src/version.rs.in src/version.rs \
                --replace-fail @COMMIT_INFO@ ""
            '';
          };
        };
      })
        (
          crate
          // {
            nativeBuildInputs = (crate.nativeBuildInputs or [ ]) ++ [
              cmake
              pkg-config
              protobuf
            ];
            buildInputs = (crate.buildInputs or [ ]) ++ [
              openssl
              openssl.dev
              zlib
            ];
          }
        );
  };
}
