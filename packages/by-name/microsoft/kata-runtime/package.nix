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
  version = "3.15.0.aks0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "microsoft";
      repo = "kata-containers";
      tag = finalAttrs.version;
      hash = "sha256-nptvqKEzxlVgMRY/7GMBoU9LbuX5V0WLGXJ9C20+zAo=";
    };

    patches = [
      # As we use a pinned version of the tardev-snapshotter per runtime version, and
      # the tardev-snapshotter's directory has a hash suffix, we must allow multiple
      # layer source directories. For now, match the layer-src-prefix with a regex.
      # We could think about moving the specific path into the settings and set it
      # to the expected value.
      #
      # This patch is not upstreamable.
      ./0001-genpolicy-regex-check-contrast-specific-layer-src-pr.patch
      # This patch builds on top of the Azure CSI patches specific to the msft
      # version of genpolicy. Therefore, we don't attempt to upstream those changes.
      # We can revisit this if microsoft upstreamed
      # https://github.com/microsoft/kata-containers/pull/174
      # TODO
      # Patches the RootfsPropagation check in allow_create_container_input to allow setting up bidirectional volumes, which need to propagate their changes to a
      # volume mounted on the root filesystem and possibly shared across multiple containers on the host.
      # RootfsPropagation describes the mapping to mount propagations: https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation
      # It reflects genpolicy-support-mount-propagation-and-ro-mounts.patch on upstream kata.genpolicy, but drops the patched propagation mode
      # derivation, because it was already built in to the microsoft fork.
      ./0002-genpolicy-support-mount-propagation-and-ro-mounts.patch
      # This patch builds on top of the Azure CSI patches specific to the msft
      # version of genpolicy. Therefore, we don't attempt to upstream those changes.
      # We can revisit this if microsoft upstreamed
      # https://github.com/microsoft/kata-containers/pull/174
      ./0003-genpolicy-support-HostToContainer-mount-propagation.patch
      # Simple genpolicy logging patch to include the image reference in case of authentication failure
      # TODO(jmxnzo): remove when authentication failure error logging includes image reference on microsoft/kata-containers fork.
      # This will be achieved when updating oci_distribution to oci_client crate on microsoft/kata-containers fork.
      # kata/kata-runtime/0011-genpolicy-bump-oci-distribution-to-v0.12.0.patch introduces this update to kata-containers.
      # After upstreaming, microsoft/kata-containers fork would need to pick up the changes.
      ./0004-genpolicy-include-reference-in-logs-when-auth-failur.patch

      # Simple genpolicy logging redaction of the policy annotation
      # This avoids printing the entire annotation on log level debug, which resulted in errors of the logtranslator.go
      # TODO(jmxnzo): remove when https://github.com/kata-containers/kata-containers/pull/10647 is picked up by microsoft/kata-containers fork
      ./0005-genpolicy-do-not-log-policy-annotation-in-debug.patch

      # Exec requests are failing on the Microsoft fork of Kata, as allow_interactive_exec is blocking execution.
      # Reason for this is that a subsequent check asserts the sandbox-name from the annotations, but such annotation
      # is only added for pods by genpolicy. The sandbox name of other pod-generating resources is hard to predict.
      #
      # With this patch, we use a regex check for the sandbox name in these cases. We construct the regex in genpolicy
      # based on the the specified metadata, following the logic after which kubernetes will derive the sandbox name.
      # The generated regex is then used in the policy to match the sandbox name.
      #
      # Microsoft was informed about the issue but didn't act since it occurred 4 months ago.
      ./0006-genpolicy-match-sandbox-name-by-regex.patch

      # Ensure that environment variables from the image configuration are not overwritten by
      # defaults in genpolicy. Fixes a regression introduced in
      # https://github.com/microsoft/kata-containers/commit/e82c19e4d5fc771bfe54b97ff0aef8a4f5c98e71.
      ./0007-genpolicy-don-t-overwrite-env-vars-from-image.patch

      # This patch fixes an issue where genpolicy can corrupt the layer cache file due to simultaneous
      # read/write operations on the file. Instead of the upstream implementation, the cache file is opened
      # read-only, changes are written to a tempfile, and the original file replaced by the tempfile atomically.
      ./0008-genpolicy-prevent-corruption-of-the-layer-cache-file.patch

      # Fix tests and build.rs so genpolicy builds.
      ./0009-fix-tests-and-protocols-build.rs.patch
    ];
  };

  sourceRoot = "${finalAttrs.src.name}/src/runtime";

  vendorHash = null;

  subPackages = [
    "cmd/containerd-shim-kata-v2"
    "cmd/kata-monitor"
    # TODO(malt3): enable kata-runtime
    # It depends on CGO and kvm
    # "cmd/kata-runtime"
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

  env.CGO_ENABLED = 0;
  ldflags = [ "-s" ];

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
})
