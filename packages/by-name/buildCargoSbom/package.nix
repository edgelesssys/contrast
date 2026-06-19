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

  mkComponent = id: {
    type = "library";
    "bom-ref" = id;
    name = (crateOf id).crateName;
    inherit ((crateOf id)) version;
    purl = purlOf (crateOf id);
  };

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
    };
    components = map mkComponent (lib.filter (id: id != member) ids);
    dependencies = map (id: {
      ref = id;
      dependsOn = dependsOn id;
    }) ids;
  };
in
# bombon consumes bombonVendoredSbom as a directory of CycloneDX files, so emit
# the SBOM into a directory rather than as a bare file.
writeTextDir "${pname}.cdx.json" (builtins.toJSON bom)
