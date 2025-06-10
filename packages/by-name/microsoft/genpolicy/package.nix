# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
  version = "3.2.0.azl5";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "microsoft";
      repo = "kata-containers";
      tag = "${version}";
      hash = "sha256-yyqVQ8EPHbhwkm2OmmgyCAFbEca5pZRjHRmlGRv9PG0=";
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

      # Exec requests are failing on the Microsoft fork of Kata, as allow_interactive_exec is blocking execution.
      # Reason for this is that a subsequent check asserts the sandbox-name from the annotations, but such annotation
      # is only added for pods by genpolicy. The sandbox name of other pod-generating resources is hard to predict.
      #
      # With this patch, we use a regex check for the sandbox name in these cases. We construct the regex in genpolicy
      # based on the the specified metadata, following the logic after which kubernetes will derive the sandbox name.
      # The generated regex is then used in the policy to match the sandbox name.
      #
      # Microsoft was informed about the issue but didn't act since it occurred 4 months ago.
      ./0011-genpolicy-match-sandbox-name-by-regex.patch

      # Fail when layer can't be processed
      # Cherry-pick from https://github.com/kata-containers/kata-containers/pull/10925,
      # which isn't yet included in the Microsoft fork.
      ./0012-genpolicy-fail-when-layer-can-t-be-processed.patch

      # Ensure that environment variables from the image configuration are not overwritten by
      # defaults in genpolicy. Fixes a regression introduced in
      # https://github.com/microsoft/kata-containers/commit/e82c19e4d5fc771bfe54b97ff0aef8a4f5c98e71.
      ./0013-genpolicy-don-t-overwrite-env-vars-from-image.patch

      # Adds the --config-file flag to genpolicy, allowing for more than one ConfigMap and/or Secret to be passed to
      # the tool.
      # This is a backport of https://github.com/kata-containers/kata-containers/pull/10986.
      ./0014-genpolicy-support-arbitrary-resources-with-c.patch

      # Ensure that binaryData fields in ConfigMaps are represented correctly as a String for
      # base64 instead of a Vec<u8>.
      # Taken from https://github.com/kata-containers/kata-containers/commit/c06bf2e3bb696016be0357c373b0c68e10602b57.
      ./0015-genpolicy-correctly-represent-binaryData-in-ConfigMa.patch

      # This patch fixes an issue where genpolicy can corrupt the layer cache file due to simultaneous
      # read/write operations on the file. Instead of the upstream implementation, the cache file is opened
      # read-only, changes are written to a tempfile, and the original file replaced by the tempfile atomically.
      ./0016-genpolicy-prevent-corruption-of-the-layer-cache-file.patch
    ];
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  useFetchCargoVendor = true;
  cargoHash = "sha256-lmRDFZ2BpTPfRmm/ckXFDRq8cTFH/ARWDiulph+E1Lc=";

  OPENSSL_NO_VENDOR = 1;

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';

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

  # Only run library tests, the integration tests need internet access.
  cargoTestFlags = [ "--lib" ];

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

    settings = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_msft_settings_prod.patch ];
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
  };

  meta = {
    changelog = "https://github.com/microsoft/kata-containers/releases/tag/genpolicy-${version}";
    homepage = "https://github.com/microsoft/kata-containers";
    mainProgram = "genpolicy";
    licesnse = lib.licenses.asl20;
  };
}
