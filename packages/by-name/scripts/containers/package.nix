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
}
// (lib.concatMapAttrs (name: container: {
  "push-${name}" = pushOCIDir name container.outPath container.meta.tag;
}) contrastPkgs.containers)
