# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
  go_1_24 = prev.go_1_24.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "1.24.4";
      src = final.fetchurl {
        url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
        hash = "sha256-WoaoOjH5+oFJC4xUIKw4T9PZWj5x+6Zlx7P5XR3+8rQ=";
      };
    }
  );

  # Add the required extensions to the Azure CLI.
  azure-cli = prev.azure-cli.override {
    withExtensions = with final.azure-cli.extensions; [ aks-preview ];
  };

  erofs-utils = prev.erofs-utils.overrideAttrs (prevAttrs: {
    # The build environment sets SOURCE_DATE_EPOCH to 1980, but as mkfs.erofs
    # implements timestamp clamping, and files from the store have a 1970
    # timestamp, we end up with different file metadata in the image
    # (in addition, it is not reproducible which files are touched during
    # the build). We cannot use the -T flag as env has precedence over
    # the flag. We therefore wrap the binary to set SOURCE_DATE_EPOCH to 0.
    nativeBuildInputs = prevAttrs.nativeBuildInputs or [ ] ++ [ final.makeWrapper ];
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
  composefs = prev.composefs.overrideAttrs (prevAttrs: {
    patches = prevAttrs.patches or [ ] ++ [
      (final.fetchpatch {
        url = "https://patch-diff.githubusercontent.com/raw/composefs/composefs/pull/415.patch";
        hash = "sha256-nzUENLM24G6NezhPywVsRzRgWmL1VZdMfZTsXNorJl8=";
      })
    ];
  });

  # Pad with zero bytes instead of zero ascii characters.
  # https://github.com/microsoft/igvm-tooling/pull/59
  igvm-tooling = prev.igvm-tooling.overrideAttrs (prevAttrs: {
    patches = prevAttrs.patches or [ ] ++ [
      (final.fetchpatch {
        name = "0002-pad-with-zero.patch";
        url = "https://github.com/microsoft/igvm-tooling/commit/f46b3b297d87ae8f11935f08cc63bcb280c4b132.patch";
        hash = "sha256-v1VBUSfQWOgqQKFoUMCl72IclirNEP8mRWVhLgKpBXY=";
        stripLen = 1;
      })
    ];
  });

  # Pinned edk2 version for OVMF-TDX.
  # TODO(katexochen): Fix OVMF-TDX measurements for newer edk2 versions.
  edk2-202411 =
    (prev.edk2.overrideAttrs (
      finalAttrs: _prevAttrs: {
        version = "202411";
        __intentionallyOverridingVersion = true; # We override srcWithVendoring instead of src.
        srcWithVendoring = final.fetchFromGitHub {
          owner = "tianocore";
          repo = "edk2";
          tag = "edk2-stable${finalAttrs.version}";
          fetchSubmodules = true;
          hash = "sha256-KYaTGJ3DHtWbPEbP+n8MTk/WwzLv5Vugty/tvzuEUf0=";
        };
      }
    )).override
      {
        buildPackages = final.buildPackages // {
          edk2 = final.edk2-202411;
          openssl = final.openssl_3;
        };
      };

  # Fix version mismatch between libnvidia-container and nvidia-container-toolkit.
  libnvidia-container = prev.libnvidia-container.overrideAttrs (
    finalAttrs: prevAttrs: {
      version = "1.17.8";
      src = final.fetchFromGitHub {
        owner = "NVIDIA";
        repo = "libnvidia-container";
        tag = "v${finalAttrs.version}";
        hash = "sha256-OzjcYxnWjzgmrjERyPN3Ch3EQj4t1J5/TbATluoDESg=";
      };
      patches =
        [
          # Patch No. 1 needs a rebase on the new version, fetch the rebased version from the upstream PR.
          (final.replaceVars (final.fetchpatch {
            url = "https://raw.githubusercontent.com/NixOS/nixpkgs/0c909a5522020455d518a7f028e4850d55080bf6/pkgs/by-name/li/libnvidia-container/0001-ldcache-don-t-use-ldcache.patch";
            hash = "sha256-mdjWLa7kSWVaoyOSNKKt59I0XxyKO+QJEnmNCh+/pPU=";
          }) { inherit (final.addDriverRunpath) driverLink; })
        ]
        ++ (
          # Apply other patches that don't need a rebase.
          builtins.tail prevAttrs.patches or [ ]
        );
    }
  );
}
