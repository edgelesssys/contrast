# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  crane,
}:

name: dir: tag:
writeShellApplication {
  name = "push-${name}";
  runtimeInputs = [ crane ];
  text = ''
    imageName="$1"
    crane push "${dir}" "$imageName:${tag}"
  '';
}
