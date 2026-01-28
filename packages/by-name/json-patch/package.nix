# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule (finalAttrs: {
  pname = "json-patch";
  version = "5.9.11";

  src = fetchFromGitHub {
    owner = "evanphx";
    repo = "json-patch";
    tag = "v${finalAttrs.version}";
    hash = "sha256-lRgz3Bw2mwQSfXvXmKUcWfexEf3YHBFy47tqWB6lzWs=";
  };

  modRoot = "v5";

  vendorHash = "sha256-W6XVd68MS0ungMgam8jefYMVhyiN6/DB+bliFzs2rdk=";

  ldflags = [ "-s" ];

  meta = {
    description = "A Go library to apply RFC6902 patches and create and apply RFC7386 patches";
    homepage = "https://github.com/evanphx/json-patch";
    license = lib.licenses.bsd3;
    mainProgram = "json-patch";
  };
})
