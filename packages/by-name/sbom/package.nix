# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildBom,
  linkFarm,
  runCommand,
  jq,
  nix,
  stdenv,
  matrix,
  contrastPkgs,
  contrastPkgsStatic,
}:

let
  inherit (lib.contrast)
    collectDerivations
    buildableOn
    vendoredSbomsOf
    supplier
    org
    ;

  version = lib.fileContents ../../../version.txt;

  withMetadata =
    id: target: raw:
    runCommand "${id}.cdx.json"
      {
        nativeBuildInputs = [
          jq
          nix
        ];
        exportReferencesGraph = [
          "closure"
          target
        ];
      }
      ''
        deployable='{}'
        while IFS= read -r ref; do
          path="/nix/store/$ref"
          [ -e "$path" ] || continue
          hash=$(nix-hash --type sha256 "$path")
          deployable=$(jq --arg r "$ref" --arg h "$hash" '. + { ($r): $h }' <<<"$deployable")
        done < <(jq -r '.components[]? | select((.purl // "") | startswith("pkg:nix")) | .["bom-ref"] // empty' ${raw})

        jq \
          --arg id ${lib.escapeShellArg id} \
          --arg version ${lib.escapeShellArg version} \
          --argjson supplier ${lib.escapeShellArg (builtins.toJSON supplier)} \
          --arg ownSource ${lib.escapeShellArg org} \
          --argjson deployable "$deployable" \
          -f ${./finish.jq} \
          ${raw} > "$out"
      '';

  # Plain-identifier product subject per category (full SBOM is "contrast").
  categoryId = {
    cli = "contrast-cli";
    runtimeConfidential = "contrast-runtime-confidential";
    runtimeNonConfidential = "contrast-runtime-non-confidential";
  };

  mkSbom =
    name: roots:
    let
      entries = buildableOn stdenv.hostPlatform (collectDerivations [ ] roots);
      target = linkFarm "contrast-${name}-matrix" entries;
    in
    withMetadata categoryId.${name} target (
      buildBom target {
        extraPaths = vendoredSbomsOf entries;
      }
    );

  categories = {
    cli = {
      inherit (contrastPkgs.contrast) cli;
      # The CLI go:embeds the (static) genpolicy binary. Embedded bytes carry no store-path reference, so it never appears in the CLI's runtime closure.
      # Adding genpolicy as an explicit root pulls in its Rust component and vendored SBOM.
      inherit (contrastPkgsStatic.kata) genpolicy;
    };
    runtimeConfidential = {
      inherit (contrastPkgs.contrast) coordinator initializer;
      inherit (contrastPkgs.kata) image agent;
      guestSystem = contrastPkgs.kata.image.passthru.toplevel;
      inherit (contrastPkgs)
        OVMF-SNP
        OVMF-TDX
        initdata-processor
        imagepuller
        imagestore
        service-mesh
        ;
    };
    runtimeNonConfidential = {
      inherit (contrastPkgs.contrast) nodeinstaller;
      inherit (contrastPkgs.kata) runtime runtime-rs;
    };
  };
in

(withMetadata "contrast" matrix (
  buildBom matrix {
    extraPaths = matrix.vendoredSbomRoots;
  }
)).overrideAttrs
  (prev: {
    passthru = (prev.passthru or { }) // lib.mapAttrs mkSbom categories;
  })
