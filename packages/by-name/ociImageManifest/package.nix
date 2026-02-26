# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# application/vnd.oci.image.manifest.v1+json
{
  lib,
  ociImageConfig,
  runCommandLocal,
  writers,
  nix,
}:
{
  # layers is a list of ociLayerTar
  layers ? [ ],
  # extraConfig is a set of extra configuration options
  extraConfig ? { },
  # extraManifest is a set of extra manifest options
  extraManifest ? { },
}:

let
  config = ociImageConfig { inherit layers extraConfig; };
  configDescriptor = builtins.fromJSON (builtins.readFile (config + "/media-descriptor.json"));
  configPlatform = builtins.fromJSON (builtins.readFile (config + "/platform.json"));
  layerDescriptors = lib.lists.map (
    layer: builtins.fromJSON (builtins.readFile (layer + "/media-descriptor.json"))
  ) layers;
  manifest = writers.writeJSON "image-manifest.json" (
    {
      schemaVersion = 2;
      mediaType = "application/vnd.oci.image.manifest.v1+json";
    }
    // extraManifest
    // {
      config = configDescriptor;
      layers = layerDescriptors;
    }
  );
in

runCommandLocal "oci-image-manifest"
  {
    blobDirs = lib.lists.map (layer: layer + "/blobs/sha256") (layers ++ [ config ]);
    platformJSON = builtins.toJSON configPlatform;
    buildInputs = [ nix ];
    inherit manifest;
  }
  ''
    mkdir -p $out/blobs/sha256
    sha256=$(nix-hash --type sha256 --flat $manifest)
    cp $manifest "$out/blobs/sha256/$sha256"
    ln -s "$out/blobs/sha256/$sha256" "$out/image-manifest.json"
    echo -n "{\"mediaType\": \"application/vnd.oci.image.manifest.v1+json\", \"size\": $(stat -c %s $manifest), \"digest\": \"sha256:$sha256\", \"platform\": $platformJSON}" > $out/media-descriptor.json
    for src in $blobDirs; do
      for blob in $(ls $src); do
        ln -s "$src/$blob" "$out/blobs/sha256/$blob"
      done
    done
  ''
