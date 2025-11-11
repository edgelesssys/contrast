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
  closure = builtins.filter (
    p: p != "" && p != configJson.outPath && !(lib.elem p (map (x: x.outPath) copyToRoot))
  ) (lib.splitString "\n" (builtins.readFile (writeClosure ([ configJson ] ++ copyToRoot))));
  layer = ociLayerTar {
    files =
      (map (pkg: {
        source = pkg;
        destination = "/";
      }) copyToRoot)
      ++ (map (path: {
        source = path;
      }) closure);
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
