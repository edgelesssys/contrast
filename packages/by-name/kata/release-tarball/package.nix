# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runtime,
  fetchzip,
}:

let
  inherit (runtime) version;
in

fetchzip {
  url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
  hash = "sha256-sEdySOuqf1WUoy+fgQzV4/mZ3zb90lTTY7s43+6oXMM=";
  stripRoot = false;
  passthru.version = version;
  passthru.updateScript = ./update.sh;
}
