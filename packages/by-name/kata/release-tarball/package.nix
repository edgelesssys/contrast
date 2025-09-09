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
  hash = "sha256-vPktGr544XWJ8vUwj0VADItFOhvZFj2owVEVPe640GM=";
  stripRoot = false;
  passthru.version = version;
  passthru.updateScript = ./update.sh;
}
