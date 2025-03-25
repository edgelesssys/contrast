# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenv,
  stdenvNoCC,
  microsoft,
  igvm-tooling,
  igvm-signing-keygen,
  igvmmeasure,
  igvm-go,
  calculateSnpIDBlock,

  debug ? false,
}:

let
  igvm-snakeoil = stdenv.mkDerivation rec {
    pname = "kata-igvm${lib.optionalString debug "-debug"}";
    inherit (microsoft.genpolicy) src version;

    nativeBuildInputs = [ igvm-tooling ];

    sourceRoot = "${src.name}/tools/osbuilder/igvm-builder";

    postPatch = ''
      chmod +x igvm_builder.sh
      substituteInPlace igvm_builder.sh \
        --replace-fail '#!/usr/bin/env bash' '#!${stdenv.shell}'

      substituteInPlace azure-linux/igvm_lib.sh \
        --replace-fail 'python3 ''${IGVM_PY_FILE}' igvmgen \
        --replace-fail '-svn $SVN' '-svn $SVN -sign_key ${igvm-signing-keygen.snakeoilPem} -sign_deterministic true' \
        --replace-fail '"''${SCRIPT_DIR}/../root_hash.txt"' ${microsoft.kata-image.verity}/dm_verity.txt

      substituteInPlace azure-linux/config.sh \
        --replace-fail '"''${IGVM_EXTRACT_FOLDER}/src/igvm/acpi/acpi-clh/"' '"${igvm-tooling}/share/igvm-tooling/acpi/acpi-clh/"' \
        --replace-fail '"''${IGVM_EXTRACT_FOLDER}/src/igvm/igvmgen.py"' '"${igvm-tooling}/bin/igvmgen"' \
        --replace-fail rootfstype=ext4 rootfstype=erofs \
        --replace-fail rootflags=data=ordered,errors=remount-ro "" \
        --replace-fail /usr/share/cloud-hypervisor/bzImage ${microsoft.kata-kernel-uvm}/bzImage
    '';

    buildPhase = ''
      runHook preBuild

      bash -x ./igvm_builder.sh -s 0 -o .

      mv kata-containers-igvm${lib.optionalString debug "-debug"}.img $out

      runHook postBuild
    '';

    dontPatchELF = true;

    passthru = {
      inherit snp-launch-digest snpIDBlock;
    };
  };

  snp-launch-digest = stdenvNoCC.mkDerivation {
    name = "launch-digest";
    dontUnpack = true;
    buildInputs = [ igvmmeasure ];
    buildPhase = ''
      mkdir $out
      igvmmeasure ${igvm-snakeoil} measure -b | dd conv=lcase > $out/milan.hex
      # Remove the trailing newline
      truncate -s -1 $out/milan.hex
    '';
  };

  snpIDBlock = calculateSnpIDBlock { inherit snp-launch-digest; };

  igvm = stdenv.mkDerivation {
    inherit (igvm-snakeoil)
      pname
      version
      src
      sourceRoot
      dontPatchELF
      ;

    nativeBuildInputs = [ igvm-go ];

    buildPhase = ''
      igvm modify ${igvm-snakeoil} \
        --snp-id-block ${snpIDBlock}/id-block-igvm-milan.json \
        --output $out
    '';

    passthru = {
      inherit snp-launch-digest snpIDBlock igvm-snakeoil;

    };

    meta = {
      description = "The Contrast runtime IGVM file defines the initial state of a pod-VM.";
      license = lib.licenses.asl20;
    };
  };
in
igvm
