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
  go_1_22 = prev.go_1_22.overrideAttrs (finalAttrs: _prevAttrs: {
    version = "1.22.2";
    src = final.fetchurl {
      url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
      hash = "sha256-N06oKyiexzjpaCZ8rFnH1f8YD5SSJQJUeEsgROkN9ak=";
    };
  });

  # Add the required extensions to the Azure CLI.
  azure-cli = prev.azure-cli.override {
    withExtensions = with final.azure-cli.extensions; [
      aks-preview
    ];
  };
}
