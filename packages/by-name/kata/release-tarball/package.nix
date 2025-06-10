# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  kata,
  fetchzip,
}:

let
  inherit (kata.kata-runtime) version;
in

fetchzip {
  url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
  hash = "sha256-6DNJ2OMTJuyS5kKW8XDz7JUNL3c7RfcsDy3J1sk+yXg=";
  stripRoot = false;
  passthru.version = version;
}
