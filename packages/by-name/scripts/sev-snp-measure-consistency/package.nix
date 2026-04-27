# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  writeShellApplication,
  contrastPkgs,
  python3Packages,
}:

let
  inherit (contrastPkgs) contrast kata;

  vcpuCounts = lib.range 1 220;
  images = [
    {
      name = "os-image";
      image = contrast.node-installer-image.os-image;
    }
    {
      name = "gpu-image";
      image = contrast.node-installer-image.gpu.os-image;
    }
  ];

  combinations = lib.concatLists (
    map (
      img:
      map (vcpus: {
        inherit (img) name;
        inherit vcpus;
        go = kata.calculateSnpLaunchDigest {
          os-image = img.image;
          inherit vcpus;
          inherit (contrast.node-installer-image) withDebug;
        };
        python = kata.calculateSnpLaunchDigest {
          os-image = img.image;
          inherit (python3Packages) sev-snp-measure;
          inherit vcpus;
          inherit (contrast.node-installer-image) withDebug;
        };
      }) vcpuCounts
    ) images
  );

  checkCmds = map (c: ''
    echo -n "Checking ${c.name} (${toString c.vcpus} vCPUs): "
    failed_local=0
    if ! diff -q "${c.python}/milan.hex" "${c.go}/milan.hex" > /dev/null; then
      failed_local=1
    fi
    if ! diff -q "${c.python}/genoa.hex" "${c.go}/genoa.hex" > /dev/null; then
      failed_local=1
    fi

    if [ $failed_local -eq 0 ]; then
      echo "OK"
    else
      echo ""
      echo "Milan diff:"
      diff  -d "${c.python}/milan.hex" "${c.go}/milan.hex" || true
      echo "Genoa diff:"
      diff -d "${c.python}/genoa.hex" "${c.go}/genoa.hex" || true
      exit_code=1
    fi
  '') combinations;

in
writeShellApplication {
  name = "verify-snp-measure";
  text = ''
    exit_code=0
    ${lib.concatStringsSep "\n" checkCmds}
    exit $exit_code
  '';
}
