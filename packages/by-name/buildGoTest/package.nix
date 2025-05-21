# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

args':

let
  args = args' // {
    "doCheck" = false;
  };
in
buildGoModule (
  {
    # copy of buildGoModule.buildPhase with the following changes:
    # - use `go test -c -o $GOPATH/bin/` instead of `go install` to build the binary of a test package
    # original:
    # https://github.com/NixOS/nixpkgs/blob/c44815411ae47dd8bbbb92d60c3a83abff28a9f3/pkgs/build-support/go/module.nix#L188-L266
    buildPhase = ''
      runHook preBuild

      exclude='\(/_\|examples\|Godeps\|testdata'
      if [[ -n "$excludedPackages" ]]; then
        IFS=' ' read -r -a excludedArr <<<$excludedPackages
        printf -v excludedAlternates '%s\\|' "''${excludedArr[@]}"
        excludedAlternates=''${excludedAlternates%\\|} # drop final \| added by printf
        exclude+='\|'"$excludedAlternates"
      fi
      exclude+='\)'

      buildGoDir() {
        local cmd="$1" dir="$2"

        . $TMPDIR/buildFlagsArray

        declare -a flags
        flags+=($buildFlags "''${buildFlagsArray[@]}")
        flags+=(''${tags:+-tags=''${tags// /,}})
        flags+=(''${ldflags:+-ldflags="$ldflags"})
        flags+=("-p" "$NIX_BUILD_CORES")

        if [[ "$cmd" = "test" ]]; then
          flags+=(-vet=off)
          flags+=($checkFlags)
        fi

        local OUT
        if ! OUT="$(go $cmd -c -o $GOPATH/bin/ "''${flags[@]}" $dir 2>&1)"; then
          if ! echo "$OUT" | grep -qE '(no( buildable| non-test)?|build constraints exclude all) Go (source )?files'; then
            echo "$OUT" >&2
            return 1
          fi
        fi
        if [[ -n "$OUT" ]]; then
          echo "$OUT" >&2
        fi
        return 0
      }

      getGoDirs() {
        local type;
        type="$1"
        if [[ -n "$subPackages" ]]; then
          echo "$subPackages" | sed "s,\(^\| \),\1./,g"
        else
          find . -type f -name \*$type.go -exec dirname {} \; | grep -v "/vendor/" | sort --unique | grep -v "$exclude"
        fi
      }

      if (( "''${NIX_DEBUG:-0}" >= 1 )); then
        buildFlagsArray+=(-x)
      fi

      if [[ ''${#buildFlagsArray[@]} -ne 0 ]]; then
        declare -p buildFlagsArray > $TMPDIR/buildFlagsArray
      else
        touch $TMPDIR/buildFlagsArray
      fi
      if [[ -z "$enableParallelBuilding" ]]; then
          export NIX_BUILD_CORES=1
      fi
      for pkg in $(getGoDirs ""); do
        echo "Building subPackage $pkg"
        buildGoDir test "$pkg"
      done
    '';
  }
  // args
)
