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

  # Use a newer uplosi that has fixes for private galleries.
  uplosi = prev.uplosi.overrideAttrs (prev: {
    src = final.fetchFromGitHub {
      owner = "edgelesssys";
      repo = prev.pname;
      rev = "fb292c23ed805cb4005fca41159d0f54bb0a5bcc";
      hash = "sha256-MsZ4Bl8sW1dZUB9cYPsaLtc8P8RRx4hafSbNB4vXqi4=";
    };
  });

  erofs-utils = prev.erofs-utils.overrideAttrs (prev: {
    patches = final.lib.optionals (prev ? patches) prev.patches ++ [
      ./erofs-utils-reproducibility.patch
    ];
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

  # Upstream PR is currently in staging: https://github.com/NixOS/nixpkgs/pull/349201.
  dtc = prev.dtc.overrideAttrs (prev: {
    patches = final.lib.optionals (prev ? patches) prev.patches ++ [
      (final.fetchpatch2 {
        url = "https://github.com/dgibson/dtc/commit/56a7d0cb3be5f2f7604bc42299e24d13a39c72d8.patch";
        hash = "sha256-GmAyk/K2OolH/Z8SsgwCcq3/GOlFuSpnVPr7jsy8Cs0=";
      })
    ];
  });
}
