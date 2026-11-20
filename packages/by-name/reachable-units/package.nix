# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  makeWrapper,
  writeShellApplication,
  coreutils,
  util-linux,
}:

let
  isolate = writeShellApplication {
    name = "isolate";
    runtimeInputs = [
      coreutils
      util-linux
    ];
    text = builtins.readFile ./isolate.sh;
  };
in

buildGoModule (finalAttrs: {
  pname = "reachable-units";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "tools/reachable-units")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-5BqvuvCMxJEgh6bYft+54oiMKxjXhxfcYfFaBZc+SMo=";

  sourceRoot = "${finalAttrs.src.name}/tools/reachable-units";

  nativeBuildInputs = [ makeWrapper ];

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  # The analysis must not see the host, so it always runs isolated, see isolate.sh.
  postInstall = ''
    mkdir -p "$out/libexec"
    mv "$out/bin/reachable-units" "$out/libexec/"
    makeWrapper ${lib.getExe isolate} "$out/bin/reachable-units" \
      --add-flags "$out/libexec/reachable-units" \
      --prefix PATH : ${
        lib.makeBinPath [
          coreutils
          util-linux
        ]
      }
  '';

  meta = lib.contrast.ourMeta { mainProgram = "reachable-units"; };
})
