# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenv,
  microsoft,
  igvm-tooling,
  igvm-signing-keygen,
}:

stdenv.mkDerivation rec {
  pname = "kata-igvm";
  inherit (microsoft.genpolicy) src version;

  outputs = [
    "out"
    "debug"
  ];

  nativeBuildInputs = [ igvm-tooling ];

  sourceRoot = "${src.name}/tools/osbuilder/igvm-builder";

  postPatch = ''
    chmod +x igvm_builder.sh
    substituteInPlace igvm_builder.sh \
      --replace-fail '#!/usr/bin/env bash' '#!${stdenv.shell}' \
      --replace-fail 'python3 ''${igvmgen_py_file}' igvmgen \
      --replace-fail '-svn $SVN' '-svn $SVN -sign_key ${igvm-signing-keygen.snakeoilPem} -sign_deterministic true' \
      --replace-fail '"''${script_dir}/../root_hash.txt"' ${microsoft.kata-image.verity}/dm_verity.txt \
      --replace-fail "install_igvm" ""

    substituteInPlace azure-linux/config.sh \
      --replace-fail '"''${igvm_extract_folder}/src/igvm/acpi/acpi-clh/"' '"${igvm-tooling}/share/igvm-tooling/acpi/acpi-clh/"' \
      --replace-fail rootfstype=ext4 rootfstype=erofs \
      --replace-fail rootflags=data=ordered,errors=remount-ro "" \
      --replace-fail /usr/share/cloud-hypervisor/bzImage ${microsoft.kata-kernel-uvm}/bzImage
  '';

  buildPhase = ''
    runHook preBuild

    bash -x ./igvm_builder.sh -s 0 -o .

    mv kata-containers-igvm.img $out
    mv kata-containers-igvm-debug.img $debug

    runHook postBuild
  '';

  meta = {
    description = "The Contrast runtime IGVM file defines the initial state of a pod-VM.";
    license = lib.licenses.asl20;
  };
}
