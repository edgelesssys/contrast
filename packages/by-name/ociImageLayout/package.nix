# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# OCI image layout. Can be pushed to a registry or used as a local image.
{
  lib,
  runCommandLocal,
  writers,
  nix,
}:

{
  # manifests is a list of ociImageManifest
  manifests ? [ ],
  # extraIndex is a set of additional fields to add to the index.json
  extraIndex ? { },
  passthru ? { },
}:

let
  manifestDescriptors = lib.lists.map (
    manifest: builtins.fromJSON (builtins.readFile (manifest + "/media-descriptor.json"))
  ) manifests;
  index = writers.writeJSON "index.json" (
    {
      schemaVersion = 2;
      mediaType = "application/vnd.oci.image.index.v1+json";
    }
    // extraIndex
    // {
      manifests = manifestDescriptors;
    }
  );
in

runCommandLocal "oci-image-layout"
  {
    buildInputs = [ nix ];
    blobDirs = lib.lists.map (manifest: manifest + "/blobs/sha256") manifests;
    inherit index passthru;
  }
  ''
    # add the index.json, oci-layout file and all blobs to the output
    srcs=($blobDirs)
    mkdir -p $out/blobs/sha256
    cp $index $out/index.json
    echo '{"imageLayoutVersion": "1.0.0"}' > $out/oci-layout
    for src in $srcs; do
      for blob in $(ls $src); do
        ln -s "$(realpath $src/$blob)" "$out/blobs/sha256/$blob"
      done
    done
  ''
