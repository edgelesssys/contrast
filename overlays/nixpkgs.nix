# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

final: prev:

{
  # Use when a version of Go is needed that is not available in the nixpkgs yet.
  # go_1_xx = prev.go_1_xx.overrideAttrs (finalAttrs: _prevAttrs: {
  #   version = "";
  #   src = final.fetchurl {
  #     url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
  #     hash = "";
  #   };
  # });

  # Add the required extensions to the Azure CLI.
  azure-cli = prev.azure-cli.override {
    withExtensions = with final.azure-cli.extensions; [ aks-preview ];
  };

  erofs-utils = prev.erofs-utils.overrideAttrs (prev: {
    # The build environment sets SOURCE_DATE_EPOCH to 1980, but as mkfs.erofs
    # implements timestamp clamping, and files from the store have a 1970
    # timestamp, we end up with different file metadata in the image
    # (in addition, it is not reproducible which files are touched during
    # the build). We cannot use the -T flag as env has precedence over
    # the flag. We therefore wrap the binary to set SOURCE_DATE_EPOCH to 0.
    nativeBuildInputs = prev.nativeBuildInputs ++ [ final.makeWrapper ];
    postFixup = ''
      wrapProgram $out/bin/mkfs.erofs \
        --set SOURCE_DATE_EPOCH 0
    '';
  });

  # Tests of composefs will detect hardware capabilities to select executed tests,
  # that's why they fail on our CI runners (Ubuntu kernel has fs-verity enabled),
  # but succeed on NixOS/Hydra.
  # See: https://github.com/composefs/composefs/pull/415
  # We need to rebuild composefs anyway as it depends on the overridden erofs-utils.
  composefs = prev.composefs.overrideAttrs (prev: {
    patches = prev.patches or [ ] ++ [
      (final.fetchpatch {
        url = "https://patch-diff.githubusercontent.com/raw/composefs/composefs/pull/415.patch";
        hash = "sha256-nzUENLM24G6NezhPywVsRzRgWmL1VZdMfZTsXNorJl8=";
      })
    ];
  });

  # A change in vale popped up several hundred new findings, likely the bug
  # described in https://github.com/errata-ai/vale/issues/955.
  # Wait for the v3.9.5 release.
  vale = prev.vale.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "3.9.3";
      src = final.fetchFromGitHub {
        owner = "errata-ai";
        repo = "vale";
        rev = "v${finalAttrs.version}";
        hash = "sha256-2IvVF/x8n1zvVXHAJLAFuDrw0Oi/RuQDa851SBlyRIk=";
      };
      vendorHash = "sha256-EWAgzb3ruxYqaP+owcyGDzNnkPDYp0ttHwCgNXuuTbk=";
      ldflags = [
        "-s"
        "-X main.version=${finalAttrs.version}"
      ];
    }
  );
}
