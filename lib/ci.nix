{ lib, ... }:
rec {
  fromPackageOutputs =
    flake: system:
    lib.concatMap (kind: lib.attrValues (lib.attrByPath [ kind system ] { } flake)) [
      "legacyPackages"
      "packages"
      "checks"
      "devShells"
      "formatters"
    ];

  allOutputs =
    flake: system:
    lib.filter lib.isDerivation (
      lib.unique (
        lib.concatMap (from: from flake system) [
          fromPackageOutputs
        ]
      )
    );
}
