# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  writeShellApplication,
  bash,
  openssh,
}:

buildGoModule {
  pname = "debugshell";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../tools/debugshell;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-gZcbNCevRe79wazZBMOJPexLG49lw25cOIa2UjGP/Cs=";

  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.bashPath=${lib.getExe bash}"
  ];

  postInstall =
    let
      debugshell = writeShellApplication {
        name = "debugshell";
        runtimeInputs = [ openssh ];
        text = ''
          if [[ ! -f /etc/passwd ]]; then
              echo "root:x:0:0:root:/root:/bin/bash" > /etc/passwd
          fi
          if [[ ! -f ./id_ed25519 ]]; then
              ssh-keygen -t ed25519 -f ./id_ed25519 -N ""
          fi
          ssh -p 2222 \
              -o StrictHostKeyChecking=no \
              -o UserKnownHostsFile=/dev/null \
              -i ./id_ed25519 \
              root@localhost \
              "$@"
        '';
      };
    in
    ''
      mv $out/bin/debugshell $out/bin/debugshell-server
      cp ${lib.getExe debugshell} $out/bin/debugshell
    '';

  meta.mainProgram = "debugshell-server";
}
