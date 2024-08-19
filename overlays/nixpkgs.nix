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

  pythonPackagesExtensions = prev.pythonPackagesExtensions ++ [
    (_pythonFinal: pythonPrev: {
      # Temporary fix for azure-cli https://github.com/NixOS/nixpkgs/issues/335750
      # Remove after pulling in https://github.com/NixOS/nixpkgs/pull/335225
      msal = pythonPrev.msal.overrideAttrs (oldAttrs: rec {
        version = "1.30.0";
        src = final.fetchPypi {
          inherit version;
          inherit (oldAttrs) pname;
          hash = "sha256-tL8AhQCS5GUVfYFO+iShj3iChMmkeUkQJNYpAwheovs=";
        };
      });
      # Temporary fix for azure-cli https://github.com/NixOS/nixpkgs/issues/335750
      # Remove after pulling in https://github.com/NixOS/nixpkgs/pull/335225
      msal-extensions = pythonPrev.msal-extensions.overrideAttrs (_oldAttrs: rec {
        version = "1.2.0";
        src = final.fetchFromGitHub {
          owner = "AzureAD";
          repo = "microsoft-authentication-extensions-for-python";
          rev = "refs/tags/${version}";
          hash = "sha256-javYE1XDW1yrMZ/BLqIu/pUXChlBZlACctbD2RfWuis=";
        };
      });
    })
  ];
}
