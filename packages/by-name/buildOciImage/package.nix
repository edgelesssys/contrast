# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# This is a shim layer to give ociImageLayout a similar interface to dockerTools.buildImage.

{
  lib,
  writers,
  writeClosure,
  ociLayerTar,
  ociImageManifest,
  ociImageLayout,
}:

{
  name,
  tag,
  copyToRoot ? [ ],
  config ? { },
}:

let
  configJson = writers.writeJSON "config.json" config;
  # All dependencies from packages in copyToRoot and nix store references in the config file.
  closureFile = writeClosure ([ configJson ] ++ copyToRoot);
  # writeClosure produces a file with one store path per line.
  closure = lib.splitString "\n" (builtins.readFile closureFile);
  # Filter out empty lines and the config file itself, which is part of the OCI directory and not needed in the layer.
  filteredClosure = builtins.filter (path: path != "" && path != configJson.outPath) closure;

  layer = ociLayerTar {
    files =
      (map (pkg: {
        source = pkg;
        destination = "/";
      }) copyToRoot)
      ++ (map (path: {
        source = path;
      }) filteredClosure);
  };

  manifest = ociImageManifest {
    layers = [ layer ];
    extraConfig = {
      inherit config;
    };
    extraManifest = {
      annotations = {
        "org.opencontainers.image.title" = name;
        "org.opencontainers.image.version" = tag;
      };
    };
  };
in

ociImageLayout {
  manifests = [ manifest ];
  passthru.meta = {
    inherit tag;
  };
}
