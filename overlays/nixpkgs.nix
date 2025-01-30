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

  # Fixes a dangling symlink in the libnvidia-container package that confuses
  # the nvidia-container-toolkit.
  # TODO(msanft): Remove once https://github.com/NixOS/nixpkgs/pull/375291 is merged and
  # pulled into Contrast.
  libnvidia-container = prev.libnvidia-container.overrideAttrs (prev: {
    postFixup = ''
      # Recreate library symlinks which ldconfig would have created
      for lib in libnvidia-container libnvidia-container-go; do
        rm -f "$out/lib/$lib.so"
        ln -s "$out/lib/$lib.so.${prev.version}" "$out/lib/$lib.so.1"
        ln -s "$out/lib/$lib.so.1" "$out/lib/$lib.so"
      done
    '';
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
