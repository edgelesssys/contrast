# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchFromGitHub,
  applyPatches,
  stdenvNoCC,
  rustPlatform,
  openssl,
  pkg-config,
  libiconv,
  zlib,
  cmake,
  protobuf,
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  version = "3.2.0.azl1.genpolicy1";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "microsoft";
      repo = "kata-containers";
      rev = "refs/tags/${version}";
      hash = "sha256-7qhwp23u3K8JEtpfhk7c1k1/n37LG7BGx8JJOEFjcXc=";
    };

    patches = [
      # Backport of https://github.com/kata-containers/kata-containers/pull/9706,
      # remove when picked up by Microsoft/kata-containers fork.
      ./0001-genpolicy-add-rules-and-types-for-volumeDevices.patch
      # Backport of https://github.com/kata-containers/kata-containers/pull/9725,
      # remove when picked up by Microsoft/kata-containers fork.
      ./0002-genpolicy-add-ability-to-filter-for-runtimeClassName.patch
      # Backport of https://github.com/kata-containers/kata-containers/pull/9864
      # remove when picked up by Microsoft/kata-containers fork.
      ./0003-genpolicy-allow-specifying-layer-cache-file.patch
      # As we use a pinned version of the tardev-snapshotter per runtime version, and
      # the tardev-snapshotter's directory has a hash suffix, we must allow multiple
      # layer source directories. For now, match the layer-src-prefix with a regex.
      # We could think about moving the specific path into the settings and set it
      # to the expected value.
      #
      # This patch is not upstreamable.
      ./0004-genpolicy-regex-check-contrast-specific-layer-src-pr.patch
      # This patch builds on top of the Azure CSI patches specific to the msft
      # version of genpolicy. Therefore, we don't attempt to upstream those changes.
      # We can revisit this if microsoft upstreamed
      # https://github.com/microsoft/kata-containers/pull/174
      ./0005-genpolicy-propagate-mount_options-for-empty-dirs.patch
      # This patch builds on top of the Azure CSI patches specific to the msft
      # version of genpolicy. Therefore, we don't attempt to upstream those changes.
      # We can revisit this if microsoft upstreamed
      # https://github.com/microsoft/kata-containers/pull/174
      ./0006-genpolicy-support-HostToContainer-mount-propagation.patch
      # This patch is a port of https://github.com/kata-containers/kata-containers/pull/10136/files
      # to Microsofts genpolicy.
      # TODO(miampf): remove when picked up by microsoft/kata-containers fork.
      ./0007-genpolicy-support-for-VOLUME-definition-in-container.patch

      # Simple genpolicy logging patch to include the image reference in case of authentication failure
      # TODO(jmxnzo): remove when authentication failure error logging includes image reference on microsoft/kata-containers fork.
      # This will be achieved when updating oci_distribution to oci_client crate on microsoft/kata-containers fork.
      # kata/kata-runtime/0011-genpolicy-bump-oci-distribution-to-v0.12.0.patch introduces this update to kata-containers.
      # After upstreaming, microsoft/kata-containers fork would need to pick up the changes.
      ./0008-genpolicy-include-reference-in-logs-when-auth-failur.patch

      # Simple genpolicy logging redaction of the policy annotation
      # This avoids printing the entire annotation on log level debug, which resulted in errors of the logtranslator.go
      # TODO(jmxnzo): remove when https://github.com/kata-containers/kata-containers/pull/10647 is picked up by microsoft/kata-containers fork
      ./0009-genpolicy-do-not-log-policy-annotation-in-debug.patch
      # Patches the RootfsPropagation check in allow_create_container_input to allow setting up bidirectional volumes, which need to propagate their changes to a
      # volume mounted on the root filesystem and possibly shared across multiple containers on the host.
      # RootfsPropagation describes the mapping to mount propagations: https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation
      # It reflects genpolicy-support-mount-propagation-and-ro-mounts.patch on upstream kata.genpolicy, but drops the patched propagation mode
      # derivation, because it was already built in to the microsoft fork.
      ./0010-genpolicy-support-mount-propagation-and-ro-mounts.patch
    ];
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  useFetchCargoVendor = true;
  cargoHash = "sha256-1qx7MfQxFaajxcG/A1Zd3L74s90AbI0tQkR6esM18xs=";

  OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    pkg-config
    cmake
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    libiconv
    zlib
  ];

  preBuild = ''
    make src/version.rs
  '';

  passthru = rec {
    settings-base = stdenvNoCC.mkDerivation {
      name = "${pname}-${version}-settings";
      inherit src sourceRoot;

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D genpolicy-settings.json $out/genpolicy-settings.json
        runHook postInstall
      '';
    };

    settings = settings-base;

    settings-coordinator = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_msft_settings_coordinator.patch ];
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_msft_settings_dev.patch ];
    };

    rules = stdenvNoCC.mkDerivation {
      name = "${pname}-${version}-rules";
      inherit src sourceRoot;

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D rules.rego $out/genpolicy-rules.rego
        runHook postInstall
      '';
    };

    rules-coordinator = applyPatches {
      src = rules;
      patches = [ ./genpolicy_msft_rules_coordinator.patch ];
    };
  };

  meta = {
    changelog = "https://github.com/microsoft/kata-containers/releases/tag/genpolicy-${version}";
    homepage = "https://github.com/microsoft/kata-containers";
    mainProgram = "genpolicy";
    licesnse = lib.licenses.asl20;
  };
}
