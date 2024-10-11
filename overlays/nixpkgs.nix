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

  inspector-foo = final.writeShellApplication {
    name = "inspector-foo";
    text = ''
      d=$(date +%s)
      echo "$@" > "$TMPDIR/inspector-foo-$d.log"
      for o in "$@"; do
        echo "$o" >> "$TMPDIR/inspector-foo-$d.log"
        if [[ $o != /* ]]; then
          continue
        fi
        if [[ -f $o ]]; then
          sha256sum "$o" >> "$TMPDIR/inspector-foo-$d.log"
        fi
        if [[ -d $o ]]; then
          find "$o" -type f | sort -k2 | xargs stat >> "$TMPDIR/inspector-foo-$d.log" || true
        fi
      done
    '';
  };

  inspector-foo-post = final.writeShellApplication {
    name = "inspector-foo-post";
    text = ''
      echo "inspector-foo-post"
    '';
  };

  erofs-utils = prev.erofs-utils.overrideAttrs (
    finalAttrs: prevAttrs: {
      nativeBuildInputs = prevAttrs.nativeBuildInputs ++ [ final.makeWrapper ];
      postFixup = ''
        wrapProgram $out/bin/mkfs.erofs
        substituteInPlace $out/bin/mkfs.erofs \
          --replace-fail exec '${final.inspector-foo}/bin/inspector-foo "$@" ; exec'
      '';
    }
  );
}
