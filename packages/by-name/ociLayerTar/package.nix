# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# application/vnd.oci.image.layer.v1.tar
# application/vnd.oci.image.layer.v1.tar+gzip
# application/vnd.oci.image.layer.v1.tar+zstd
{
  lib,
  runCommandLocal,
  nix,
  rsync,
  pigz,
  zstd,
}:
{
  # files is a list of objects with the following attributes:
  #   source: the path to the file or directory to include in the layer
  #   destination: the path to place the file or directory in the layer
  files ? [ ],
  # compression is the compression algorithm to use, either "gzip" or "zstd"
  compression ? "gzip",
}:

runCommandLocal "ociLayer"
  {
    fileSources = lib.lists.map (file: file.source) files;
    fileDestinations = lib.lists.map (file: file.destination or file.source) files;
    outPath =
      "layer"
      + (
        if compression == "gzip" then
          ".tar.gz"
        else if compression == "zstd" then
          ".tar.zst"
        else
          ".tar"
      );
    mediaType =
      "application/vnd.oci.image.layer.v1.tar" + (if compression == "" then "" else "+" + compression);
    nativeBuildInputs = [
      nix
      rsync
    ]
    ++ lib.optional (compression == "gzip") pigz
    ++ lib.optional (compression == "zstd") zstd;
    inherit compression;
  }
  ''
    set -o pipefail
    srcs=($fileSources)
    dests=($fileDestinations)
    mkdir -p ./root $out

    echo "Adding contents..."
    # Copy files into the tree (./root/)
    for i in ''${!srcs[@]}; do
        chmod ug+w ./root
        mkdir -p "./root/$(dirname ''${dests[$i]})"
        if [ -d "''${srcs[i]}" ]; then
          # Copy contents of directory
          srcPath="''${srcs[i]}/"
        else
          srcPath="''${srcs[i]}"
        fi
        rsync -ak --chown=root:0 "$srcPath" "./root/''${dests[$i]}"
    done

    # Create the layer tarball
    echo "Packing layer..."
    tar --hard-dereference --sort=name --owner=root:0 --group=root:0 --mtime='UTC 1970-01-01' -cC ./root -f $out/layer.tar .
    # Calculate the layer tarball's diffID (hash of the uncompressed tarball)
    echo "Calculating layer tarball hash..."
    diffID=$(sha256sum $out/layer.tar | cut -d' ' -f1)
    # Compress the layer tarball
    echo "Compressing layer tarball..."
    if [[ "$compression" = "gzip" ]]; then
      pigz -c $out/layer.tar > $out/$outPath
    elif [[ "$compression" = "zstd" ]]; then
      zstd -T0 -q -c $out/layer.tar > $out/$outPath
    else
      mv $out/layer.tar $out/$outPath
    fi
    rm -f $out/layer.tar

    # Calculate the blob's sha256 hash and write the media descriptor
    echo "Calculating layer blob hash..."
    sha256=$(sha256sum $out/$outPath | cut -d' ' -f1)
    echo -n "{\"mediaType\": \"$mediaType\", \"size\": $(stat -c %s $out/$outPath), \"digest\": \"sha256:$sha256\"}" > $out/media-descriptor.json
    echo -n "sha256:$diffID" > $out/DiffID

    # Move the compressed layer tarball to the blobs directory and create a symlink
    mkdir -p $out/blobs/sha256
    mv $out/$outPath $out/blobs/sha256/$sha256
    ln -s $out/blobs/sha256/$sha256 $out/$outPath
    chmod -R ug+w ./root
    rm -rf ./root

    echo "Finished building layer"
  ''
