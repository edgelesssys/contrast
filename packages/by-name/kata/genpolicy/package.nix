# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  source,
  buildCargoSbom,
  stdenvNoCC,
  applyPatches,
}:

(source.cargoNixPackage.workspaceMembers."genpolicy".build.override {
  runTests = false;
}).overrideAttrs
  (prev: {
    pname = "genpolicy";
    passthru = (prev.passthru or { }) // rec {
      bombonVendoredSbom = buildCargoSbom {
        inherit (source) cargoNixPackage;
        member = "genpolicy";
      };

      settings-base = stdenvNoCC.mkDerivation {
        name = "genpolicy-${source.version}-settings";
        inherit (source) src;
        sourceRoot = "${source.src.name}";
        phases = [
          "unpackPhase"
          "patchPhase"
          "installPhase"
        ];
        installPhase = ''
          runHook preInstall
          install -D src/tools/genpolicy/genpolicy-settings.json $out/genpolicy-settings.json
          runHook postInstall
        '';
      };

      # We maintain two different patches for the genpolicy settings, one for development and one for
      # the release. We can't apply both to the Kata sources at the same time, so we have two
      # derivations here that apply the patches only to the settings file.
      #
      # If you need to modify these patches, this workflow may come in handy to keep diffs small.
      # Replace $CONTRAST with your repository worktree and adjust the patch file to _prod, if
      # needed.
      #
      #   cd $CONTRAST
      #   mkdir -p /tmp/a /tmp/b
      #   nix build .#base.kata.genpolicy.settings-base
      #   cp --no-preserve=mode result/genpolicy-settings.json /tmp/b
      #   cd /tmp/b
      #   patch -b -B ../a/ -p1 genpolicy-settings.json <$CONTRAST/packages/by-name/kata/genpolicy/genpolicy_settings_dev.patch
      #   # Now, edit /tmp/b/genpolicy-settings.json according to your needs.
      #   cd ..
      #   git diff --no-ext-diff --full-index --no-prefix a/genpolicy-settings.json b/genpolicy-settings.json >$CONTRAST/packages/by-name/kata/genpolicy/genpolicy_settings_dev.patch
      settings = applyPatches {
        src = settings-base;
        patches = [ ./genpolicy_settings_prod.patch ];
      };

      # Settings that allow exec into CVM pods - not safe for production use!
      settings-dev = applyPatches {
        src = settings-base;
        patches = [ ./genpolicy_settings_dev.patch ];
      };

      # Switch to rules-allow-all to disable policy checks for debugging.
      rules = rules-prod;

      rules-prod = stdenvNoCC.mkDerivation {
        name = "genpolicy-${source.version}-rules";
        inherit (source) src;
        sourceRoot = "${source.src.name}";
        phases = [
          "unpackPhase"
          "patchPhase"
          "installPhase"
        ];
        installPhase = ''
          runHook preInstall
          install -D src/tools/genpolicy/rules.rego $out/genpolicy-rules.rego
          runHook postInstall
        '';
      };

      rules-allow-all = stdenvNoCC.mkDerivation {
        name = "genpolicy-${source.version}-rules-allow-all";
        inherit (source) src;
        sourceRoot = "${source.src.name}";
        phases = [
          "unpackPhase"
          "patchPhase"
          "installPhase"
        ];
        installPhase = ''
          runHook preInstall
          install -D ../../kata-opa/allow-all.rego $out/genpolicy-rules.rego
          runHook postInstall
        '';
      };
    };
    meta = (prev.meta or { }) // {
      changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${source.version}";
      homepage = "https://github.com/kata-containers/kata-containers";
      mainProgram = "genpolicy";
      license = lib.licenses.asl20;
    };
  })
