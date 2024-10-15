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
}
