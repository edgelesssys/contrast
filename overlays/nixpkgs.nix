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

  # There is a regression in 2.1.0, so pin to 2.0.5 for now.
  # TODO(katexochen): In contact with upstream maintainers to resolve the issue.
  treefmt2 = prev.treefmt2.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "2.0.5";
      src = final.fetchFromGitHub {
        owner = "numtide";
        repo = "treefmt";
        rev = "v${finalAttrs.version}";
        hash = "sha256-lDQbrq9AWH5Hjgy5AllbLLBUl/JkYGw68M5wob14kus=";
      };
      vendorHash = "sha256-OyOgTBwcRNd6kdnn3TFuq7xukeK0A1imK/WMer0tldk=";
    }
  );
}
