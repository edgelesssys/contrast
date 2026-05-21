# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Shared source meta-package for the kata-containers fork.
#
# Owns the upstream version pin, patched source, and the workspace-wide
# `cargoNixPackage` (callPackage of the committed `./Cargo.nix`, with the
# crate2nix-style `crateOverrides` that wire native deps and source fixups
# into the per-crate builds). Consumed by `kata.agent`, `kata.runtime-rs`,
# and `kata.genpolicy` via the nested by-name scope.
#
# Regenerate `Cargo.nix` after any change to the kata `Cargo.lock` or to
# the patch set under this directory:
#
#   nix build .#base.kata.source.src
#   cp -r --no-preserve=mode,ownership result /tmp/kata-src
#   cd /tmp/kata-src
#   nix run nixpkgs#crate2nix -- generate -f Cargo.toml -o Cargo.nix
#   sed -i -e 's|^{ nixpkgs ? <nixpkgs>$|{ workspaceSrc\n, nixpkgs ? <nixpkgs>|' \
#          -e 's|src = \./|src = workspaceSrc + "/|' \
#          -e 's|src = workspaceSrc + "/\([^;]*\);|src = workspaceSrc + "/\1";|' \
#          Cargo.nix
#   # Normalize the workspace path so re-generations don't churn the diff.
#   sed -i 's|/tmp/kata-src/src/libs/safe-path|/tmp/c2n-gen/src/libs/safe-path|g' Cargo.nix
#   # safe-path appears twice (workspace + registry); collapse to the workspace one.
#   sed -i 's|"registry+https://github.com/rust-lang/crates.io-index#safe-path@0.1.0"|"path+file:///tmp/c2n-gen/src/libs/safe-path#0.1.0"|g' Cargo.nix
#   # The test-runner derivation's buildPhase writes the test log directly to
#   # `$out` via `tee`, so the default `installPhase` (which mkdir's `$out`)
#   # fails. crate2nix doesn't emit `dontInstall = true` itself — patch it in.
#   sed -i '/buildInputs = testInputs;/a\\\n            dontInstall = true;' Cargo.nix
#   cp Cargo.nix $CONTRAST/packages/by-name/kata/source/Cargo.nix
#   # then delete the leftover (now-duplicate) registry safe-path block.

{
  fetchFromGitHub,
  applyPatches,
  rustPlatform,
  lib,
  callPackage,
  buildRustCrate,
  defaultCrateOverrides,
  protobuf,
  pkg-config,
  openssl,
  libseccomp,
  lvm2,
  cmake,
  zlib,
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

  cargoNixPackage = callPackage ./Cargo.nix {
    workspaceSrc = src;
    buildRustCrateForPkgs =
      _:
      buildRustCrate.override {
        defaultCodegenUnits = 16;
        defaultCrateOverrides = defaultCrateOverrides // {
          protocols = _: {
            nativeBuildInputs = [ protobuf ];
          };
          prost-build = _: {
            nativeBuildInputs = [ protobuf ];
          };
          k8s-cri = _: {
            nativeBuildInputs = [ protobuf ];
          };
          containerd-client = _: {
            nativeBuildInputs = [ protobuf ];
          };
          openssl-sys = _: {
            nativeBuildInputs = [ pkg-config ];
            buildInputs = [
              openssl
              openssl.dev
            ];
            OPENSSL_NO_VENDOR = 1;
          };
          libseccomp-sys = _: {
            nativeBuildInputs = [ pkg-config ];
            buildInputs = [
              libseccomp
              libseccomp.dev
              libseccomp.lib
            ];
          };
          kata-agent = _: {
            nativeBuildInputs = [
              cmake
              protobuf
            ];
            buildInputs = [
              openssl
              openssl.dev
              lvm2.dev
              libseccomp
              libseccomp.dev
              libseccomp.lib
              rustPlatform.bindgenHook
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
            nativeBuildInputs = [
              pkg-config
              protobuf
            ];
            buildInputs = [
              openssl
              openssl.dev
            ];
            OPENSSL_NO_VENDOR = 1;
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
            nativeBuildInputs = [
              cmake
              pkg-config
              protobuf
            ];
            buildInputs = [
              openssl
              openssl.dev
              zlib
            ];
            OPENSSL_NO_VENDOR = 1;
            OPENSSL_DIR = "${openssl.dev}";
            OPENSSL_LIB_DIR = "${lib.getLib openssl}/lib";
            preBuild = ''
              substitute src/version.rs.in src/version.rs \
                --replace-fail @COMMIT_INFO@ ""
            '';
          };
        };
      };
  };
}
