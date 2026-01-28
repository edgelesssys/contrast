# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule {
  pname = "json-patch";
  version = "v5.9.11";

  src = fetchFromGitHub {
    owner = "evanphx";
    repo = "json-patch";
    rev = "v5.9.11";
    hash = "sha256-lRgz3Bw2mwQSfXvXmKUcWfexEf3YHBFy47tqWB6lzWs=";
  };

  modRoot = "v5";

  vendorHash = "sha256-W6XVd68MS0ungMgam8jefYMVhyiN6/DB+bliFzs2rdk=";
}
