# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Generates a CycloneDX SBOM for a crate2nix workspace member purely from the
# generated Cargo.nix dependency graph — no cargo or external SBOM tool is run.
# The crate set and features are resolved exactly as the build resolves them
# (via the Cargo.nix internal mergePackageFeatures), for the host target and
# excluding dev/test dependencies. Set
#
#   passthru.sbom = buildCargoSbom { inherit cargoNixPackage; member = "..."; };
#
# on a crate2nix package; the top-level sbom package collects it automatically.

{
  lib,
  writeTextDir,
  stdenv,
}:

{
  # The package set produced from a crate2nix Cargo.nix (must expose `internal`).
  cargoNixPackage,
  # Workspace member packageId to describe, e.g. "kata-agent".
  member,
  # Cargo features enabled for the member. Must match the features the built
  # derivation is overridden with, so the SBOM reflects what is actually built.
  features ? [ "default" ],
  pname ? member,
}:

let
  inherit (cargoNixPackage) internal;
  inherit (internal) crates;

  target = internal.makeDefaultTarget stdenv.hostPlatform // {
    test = false;
  };

  # packageId -> enabled features for the full transitive (non-dev) closure,
  # resolved the same way buildRustCrateWithFeatures resolves it.
  resolved = internal.mergePackageFeatures {
    packageId = member;
    inherit features target;
  };
  ids = builtins.attrNames resolved;
  inClosure = builtins.listToAttrs (map (id: lib.nameValuePair id true) ids);

  crateOf = id: crates.${id};
  purlOf = c: "pkg:cargo/${c.crateName}@${c.version}";

  # Author and licence metadata, mapped from the fields crate2nix emits into
  # Cargo.nix. `authors` is present today; `license` is not yet forwarded by
  # crate2nix (see the note at the bottom of this file), so the licence attr is a
  # no-op until that lands, at which point every crate gains a concluded licence.
  metaOf =
    c:
    lib.optionalAttrs ((c.authors or [ ]) != [ ]) {
      # CycloneDX 1.5 carries a single author string; join multiple authors.
      author = lib.concatStringsSep ", " c.authors;
    }
    // lib.optionalAttrs ((c.license or null) != null) {
      # Normalise the deprecated Cargo `A/B` form to the SPDX `A OR B` expression
      # (SPDX ignores the extra whitespace); some crates still use the slash form.
      licenses = [ { expression = builtins.replaceStrings [ "/" ] [ " OR " ] c.license; } ];
    };

  mkComponent =
    id:
    let
      c = crateOf id;
    in
    {
      type = "library";
      "bom-ref" = id;
      name = c.crateName;
      inherit (c) version;
      purl = purlOf c;
    }
    // metaOf c;

  rootCrate = crateOf member;

  # Edges to every resolved dependency whose target crate is also in the closure
  # (this drops disabled-optional and foreign-target dependencies).
  dependsOn =
    id:
    let
      c = crateOf id;
    in
    lib.unique (
      map (d: d.packageId) (
        lib.filter (d: inClosure ? ${d.packageId}) ((c.dependencies or [ ]) ++ (c.buildDependencies or [ ]))
      )
    );

  bom = {
    # 1.5 is what bombon's transformer parses for vendored SBOMs; bombon converts
    # the aggregated output to 1.7.
    "$schema" = "http://cyclonedx.org/schema/bom-1.5.schema.json";
    bomFormat = "CycloneDX";
    specVersion = "1.5";
    version = 1;
    metadata.component = {
      type = "application";
      "bom-ref" = member;
      name = rootCrate.crateName;
      inherit (rootCrate) version;
      purl = purlOf rootCrate;
    }
    // metaOf rootCrate;
    components = map mkComponent (lib.filter (id: id != member) ids);
    dependencies = map (id: {
      ref = id;
      dependsOn = dependsOn id;
    }) ids;
  };
in
# bombon consumes bombonVendoredSbom as a directory of CycloneDX files, so emit
# the SBOM into a directory rather than as a bare file.
#
# NOTE on licences (TR-03183-2 / CRA): crate2nix already resolves each crate's
# licence via `cargo metadata` at generation time but does not write it into
# Cargo.nix (it forwards `authors` but not `license`). Forwarding `license`
# upstream — symmetric with the existing `authors` — and regenerating Cargo.nix
# would populate a concluded licence for every crate here with no build and no
# import-from-derivation. Until then `metaOf` emits authors only.
writeTextDir "${pname}.cdx.json" (builtins.toJSON bom)
