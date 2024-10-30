# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

final: prev:
# TODO(miampf): Remove unneccessary block once https://github.com/NixOS/nixpkgs/pull/345326 is merged into unstable nixpkgs
let
  # Builder for Azure CLI extensions. Extensions are Python wheels that
  # outside of nix would be fetched by the CLI itself from various sources.
  mkAzExtension =
    {
      pname,
      url,
      sha256,
      description,
      ...
    }@args:
    prev.python3.pkgs.buildPythonPackage (
      {
        format = "wheel";
        src = prev.fetchurl { inherit url sha256; };
        meta = {
          inherit description;
          inherit (prev.azure-cli.meta) platforms maintainers;
          homepage = "https://github.com/Azure/azure-cli-extensions";
          changelog = "https://github.com/Azure/azure-cli-extensions/blob/main/src/${pname}/HISTORY.rst";
          license = prev.lib.licenses.mit;
          sourceProvenance = [ prev.lib.sourceTypes.fromSource ];
        } // args.meta or { };
      }
      // (removeAttrs args [
        "url"
        "sha256"
        "description"
        "meta"
      ])
    );

  confcom = mkAzExtension rec {
    pname = "confcom";
    version = "1.0.0";
    url = "https://azcliprod.blob.core.windows.net/cli-extensions/confcom-${version}-py3-none-any.whl";
    sha256 = "73823e10958a114b4aca84c330b4debcc650c4635e74c568679b6c32c356411d";
    description = "Microsoft Azure Command-Line Tools Confidential Container Security Policy Generator Extension";
    nativeBuildInputs = [ prev.autoPatchelfHook ];
    buildInputs = [ prev.openssl_1_1 ];
    propagatedBuildInputs = with prev.python3Packages; [
      pyyaml
      deepdiff
      docker
      tqdm
    ];
    postInstall = ''
      chmod +x $out/${prev.python3.sitePackages}/azext_confcom/bin/genpolicy-linux
    '';
    meta.maintainers = with prev.lib.maintainers; [ miampf ];
  };
in
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
    withExtensions = with final.azure-cli.extensions; [
      aks-preview
      confcom
    ];
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
  });
}
