# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  contrastPkgs,
  writeShellApplication,
}:

let
  pushOCIDir =
    name: dir: tag:
    writeShellApplication {
      name = "push-${name}";
      runtimeInputs = with pkgs; [ crane ];
      text = ''
        imageName="$1"
        containerlookup="''${2:-/dev/null}"
        layersCache="''${3:-$(mktemp)}"
        hash=$(crane push "${dir}" "$imageName:${tag}")
        printf "ghcr.io/edgelesssys/contrast/%s:latest=%s\n" "${name}" "$hash" >> "$containerlookup"
        if [ ! -f "$layersCache" ]; then
          echo -n "[]" > "$layersCache"
        fi
        jq -s 'add' "$layersCache" "${dir}/layers-cache.json" > tmp.json && mv tmp.json "$layersCache"
        echo "$hash"
      '';
    };
in
{
  push-node-installer-kata =
    pushOCIDir "node-installer-kata" contrastPkgs.contrast.node-installer-image
      "v${contrastPkgs.contrast.nodeinstaller.version}";
  push-node-installer-kata-gpu =
    pushOCIDir "node-installer-kata-gpu" contrastPkgs.contrast.node-installer-image.gpu
      "v${contrastPkgs.contrast.nodeinstaller.version}";

  # This is a script rather than a contrastPkgs.buildOciImage because we *want* impurity here,
  # in order to generate a new image digest and a new tag every time the script runs.
  push-containerd-reproducer = writeShellApplication {
    name = "push-containerd-reproducer";
    runtimeInputs = with pkgs; [
      jq
      umoci
      skopeo
    ];
    text = ''
      registry="$1"
      tmpdir="$(mktemp -d)/oci"
      timestamp=$(date +%s)

      umoci init --layout "$tmpdir"
      skopeo copy "docker://ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b" "oci:$tmpdir:alpine" --insecure-policy
      umoci unpack --image "$tmpdir:alpine" "$tmpdir/rootfs" --rootless
      echo "$timestamp" > "$tmpdir/rootfs/rootfs/timestamp"
      umoci repack --image "$tmpdir:alpine" "$tmpdir/rootfs"
      skopeo copy "oci:$tmpdir" "docker://$registry/contrast/containerd-reproducer:$timestamp" --insecure-policy

      digest=$(jq -r '.manifests[0].digest' "$tmpdir/index.json")
      echo "$timestamp $digest"
    '';
  };
}
// (lib.concatMapAttrs (name: container: {
  "push-${name}" = pushOCIDir name container.outPath container.meta.tag;
}) contrastPkgs.containers)
