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
  go_1_25 = prev.go_1_25.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "1.25.3";
      src = final.fetchurl {
        url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
        hash = "sha256-qBpLpZPQAV4QxR4mfeP/B8eskU38oDfZUX0ClRcJd5U=";
      };
    }
  );

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
      finalAttrs: prevAttrs: {
        version = "202411";
        __intentionallyOverridingVersion = true; # We override srcWithVendoring instead of src.
        srcWithVendoring = final.fetchFromGitHub {
          owner = "tianocore";
          repo = "edk2";
          tag = "edk2-stable${finalAttrs.version}";
          fetchSubmodules = true;
          hash = "sha256-KYaTGJ3DHtWbPEbP+n8MTk/WwzLv5Vugty/tvzuEUf0=";
        };
        src = prevAttrs.src.overrideAttrs (
          _srcFinalAttrs: _srcPrevAttrs: {
            patches = [
              # pass targetPrefix as an env var
              (final.fetchpatch {
                url = "https://src.fedoraproject.org/rpms/edk2/raw/08f2354cd280b4ce5a7888aa85cf520e042955c3/f/0021-Tweak-the-tools_def-to-support-cross-compiling.patch";
                hash = "sha256-E1/fiFNVx0aB1kOej2DJ2DlBIs9tAAcxoedym2Zhjxw=";
              })
              # https://github.com/tianocore/edk2/pull/5658
              (final.fetchpatch {
                name = "fix-cross-compilation-antlr-dlg.patch";
                url = "https://github.com/tianocore/edk2/commit/a34ff4a8f69a7b8a52b9b299153a8fac702c7df1.patch";
                hash = "sha256-u+niqwjuLV5tNPykW4xhb7PW2XvUmXhx5uvftG1UIbU=";
              })
            ];
          }
        );

      }
    )).override
      {
        buildPackages = final.buildPackages // {
          edk2 = final.edk2-202411;
          openssl = final.openssl_3;
        };
      };

  # The fragment checks in 0.19.1 are broken.
  # ToDO(katexochen): Check on lychee versions >0.19.1.
  lychee = prev.lychee.overrideAttrs (
    finalAttrs: _prevAttrs: {
      version = "0.18.1";
      src = final.fetchFromGitHub {
        owner = "lycheeverse";
        repo = "lychee";
        tag = "lychee-v${finalAttrs.version}";
        hash = "sha256-aT7kVN2KM90M193h4Xng6+v69roW0J4GLd+29BzALhI=";
      };
      cargoDeps = final.rustPackages.rustPlatform.fetchCargoVendor {
        inherit (finalAttrs) src pname version;
        hash = "sha256-TKKhT4AhV2uzXOHRnKHiZJusNoCWUliKmKvDw+Aeqnc=";
      };
      # The postFixup script calls the binary to generate man pages, which was only introduced in v0.21.0.
      postFixup = null;
    }
  );
}
