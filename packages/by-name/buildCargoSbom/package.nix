# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  writeTextDir,
}:

{
  cargoNixPackage,
  member,
  pname ? member,
}:

let
  inherit (cargoNixPackage.internal) crates;

  crateOf = id: crates.${id};
  purlOf = c: "pkg:cargo/${c.crateName}@${c.version}";

  depsOf =
    c:
    lib.filter (id: crates ? ${id}) (
      map (d: d.packageId) ((c.dependencies or [ ]) ++ (c.buildDependencies or [ ]))
    );
  closure = builtins.genericClosure {
    startSet = [ { key = member; } ];
    operator = { key }: map (id: { key = id; }) (depsOf (crateOf key));
  };
  ids = map (e: e.key) closure;
  inClosure = builtins.listToAttrs (map (id: lib.nameValuePair id true) ids);

  metaOf =
    c:
    let
      # Normalise the deprecated Cargo `A/B` form some crates still use to the SPDX `A OR B` expression.
      spdx =
        if (c.license or null) != null then builtins.replaceStrings [ "/" ] [ " OR " ] c.license else null;

      externalReferences =
        lib.optional ((c.repository or null) != null) {
          type = "vcs";
          url = c.repository;
        }
        ++ lib.optional ((c.homepage or null) != null) {
          type = "website";
          url = c.homepage;
        }
        ++
          lib.optional
            ((c.repository or null) == null && (c.homepage or null) == null && (c.sha256 or null) != null)
            {
              type = "website";
              url = "https://crates.io/crates/${c.crateName}";
            };

      hashes = lib.optional ((c.sha256 or null) != null) {
        alg = "SHA-256";
        content = builtins.convertHash {
          hash = c.sha256;
          hashAlgo = "sha256";
          toHashFormat = "base16";
        };
      };

      contacts = map (name: { inherit name; }) (c.authors or [ ]);
    in
    lib.optionalAttrs (contacts != [ ]) {
      supplier.contact = contacts;
    }
    // lib.optionalAttrs (spdx != null) {
      licenses = [ { expression = spdx; } ];
    }
    // lib.optionalAttrs (hashes != [ ]) { inherit hashes; }
    // lib.optionalAttrs (externalReferences != [ ]) { inherit externalReferences; };

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

  # Edges to every resolved dependency whose target crate is also in the closure.
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
    "$schema" = "http://cyclonedx.org/schema/bom-1.5.schema.json";
    bomFormat = "CycloneDX";
    specVersion = "1.5"; # 1.5 is what bombon's transformer parses for vendored SBOMs.
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
writeTextDir "${pname}.cdx.json" (builtins.toJSON bom)
