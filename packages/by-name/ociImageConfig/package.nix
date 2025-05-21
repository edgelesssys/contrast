# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# application/vnd.oci.image.config.v1+json
{
  lib,
  runCommand,
  writers,
  nix,
}:
{
  # layers is a list of ociLayerTar
  layers ? [ ],
  # extraConfig is a set of extra configuration options
  extraConfig ? { },
}:

let
  diffIDs = lib.lists.map (layer: builtins.readFile (layer + "/DiffID")) layers;
  config =
    {
      architecture = "amd64";
      os = "linux";
    }
    // extraConfig
    // {
      rootfs = {
        type = "layers";
        diff_ids = diffIDs;
      };
    };
  configJSON = writers.writeJSON "image-config.json" config;
in

runCommand "oci-image-config"
  {
    buildInputs = [ nix ];
    platformJSON = builtins.toJSON {
      inherit (config) architecture;
      inherit (config) os;
    };
    inherit configJSON;
  }
  ''
    # write the config to a file under blobs/sha256
    mkdir -p $out/blobs/sha256
    sha256=$(nix-hash --type sha256 --flat $configJSON)
    cp $configJSON "$out/blobs/sha256/$sha256"

    # create a symlink to the image config
    ln -s "$out/blobs/sha256/$sha256" "$out/image-config.json"
    # write the platform.json
    echo "$platformJSON" > "$out/platform.json"
    # write the media descriptor
    echo -n "{\"mediaType\": \"application/vnd.oci.image.config.v1+json\", \"size\": $(stat -c %s $configJSON), \"digest\": \"sha256:$sha256\"}" > $out/media-descriptor.json
  ''
