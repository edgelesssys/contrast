# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

final: prev: {
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

  # There is a reproducibility issue in later versions,
  # likely https://git.kernel.org/pub/scm/linux/kernel/git/xiang/erofs-utils.git/commit/?id=0da388cfdc9dcb952c01b0755ab8a1d6d59a5312
  erofs-utils = prev.erofs-utils.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "1.7.1";
      src = final.fetchurl {
        url = "https://git.kernel.org/pub/scm/linux/kernel/git/xiang/erofs-utils.git/snapshot/erofs-utils-${finalAttrs.version}.tar.gz";
        hash = "sha256-GWCD1j5eIx+1eZ586GqUS7ylZNqrzj3pIlqKyp3K/xU=";
      };
    }
  );
}
