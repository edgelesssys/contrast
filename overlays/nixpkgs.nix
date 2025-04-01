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

  composefs = prev.composefs.overrideAttrs (prev: {
    preCheck =
      prev.preCheck
      + ''

        fatal() {
            echo $@ 1>&2; exit 1
        }

        # Dump ls -al + file contents to stderr, then fatal()
        _fatal_print_file() {
            file="$1"
            shift
            ls -al "$file" >&2
            sed -e 's/^/# /' < "$file" >&2
            fatal "$@"
        }

        assert_file_has_content () {
            fpath=$1
            shift
            for re in "$@"; do
                if ! grep -q -e "$re" "$fpath"; then
                    _fatal_print_file "$fpath" "File '$fpath' doesn't match regexp '$re'"
                fi
            done
        }

        check_whiteout () {
            tmpfile=$(mktemp --tmpdir lcfs-whiteout.XXXXXX)
            rm -f $tmpfile
            if mknod $tmpfile c 0 0 &> /dev/null; then
                echo y
            else
                echo n
            fi
            rm -f $tmpfile
        }

        check_fuse () {
            fusermount --version >/dev/null 2>&1 || return 1

            capsh --print | grep -q 'Bounding set.*[^a-z]cap_sys_admin' || \
                return 1

            [ -w /dev/fuse ] || return 1
            [ -e /etc/mtab ] || return 1

            return 0
        }

        check_erofs_fsck () {
            if which fsck.erofs &>/dev/null; then
                echo y
            else
                echo n
            fi
        }

        check_fsverity () {
            fsverity --version >/dev/null 2>&1 || return 1
            tmpfile=$(mktemp --tmpdir lcfs-fsverity.XXXXXX)
            echo foo > $tmpfile
            fsverity enable $tmpfile >&2 1>&2  || return 1
            return 0
        }

        assert_streq () {
            if test "$1" != "$2"; then
                echo "assertion failed: $1 = $2" 1>&2
                return 1
            fi
        }

        [[ -v can_whiteout ]] || can_whiteout=$(check_whiteout)
        [[ -v has_fuse ]] || has_fuse=$(if check_fuse; then echo y; else echo n; fi)
        [[ -v has_fsck ]] || has_fsck=$(check_erofs_fsck)
        [[ -v has_fsverity ]] || has_fsverity=$(if check_fsverity; then echo y; else echo n; fi)

        echo Test options: can_whiteout=$can_whiteout has_fuse=$has_fuse has_fsck=$has_fsck has_fsverity=$has_fsverity


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
