# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, fetchFromGitHub
, stdenv
, igvm-signing-keygen
, igvm-tooling
, kata-image
, kata-kernel-uvm
}:

stdenv.mkDerivation rec {
  pname = "kata-igvm";
  # This is not a real version, since the igvm builder is not part of the official release
  version = "3.2.0.igvm";

  outputs = [ "out" "debug" ];

  nativeBuildInputs = [
    igvm-tooling
  ];

  # keep up to date with the igvm-builder branch
  # https://github.com/microsoft/kata-containers/tree/dadelan/igvm-builder
  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "ad93335ff0d1502a6f094324aa87275c8201c684";
    hash = "sha256-pVogv30WsQejBtheGz76O4MDUs1+nxm8Xr6LXGmtolg=";
  };

  sourceRoot = "${src.name}/tools/osbuilder/igvm-builder";

  postPatch = ''
    chmod +x igvm_builder.sh
    substituteInPlace igvm_builder.sh \
      --replace-fail '#!/usr/bin/env bash' '#!${stdenv.shell}' \
      --replace-fail 'python3 igvm/igvmgen.py' igvmgen \
      --replace-fail igvm/acpi/acpi-clh/ "${igvm-tooling}/share/igvm-tooling/acpi/acpi-clh/" \
      --replace-fail rootfstype=ext4 rootfstype=erofs \
      --replace-fail rootflags=data=ordered,errors=remount-ro "" \
      --replace-fail '-svn 0' '-svn 0 -sign_key ${igvm-signing-keygen.snakeoilPem} -sign_deterministic true' \
      --replace-fail 'mv ''${igvm_name} ''${script_dir}' "" \
      --replace-fail sudo ""
  '';

  buildPhase = ''
    runHook preBuild

    # prevent non-hermetic download of igvm-tooling / igvmgen
    mkdir -p msigvm-1.2.0
    ./igvm_builder.sh -k ${kata-kernel-uvm}/bzImage -v ${kata-image.verity}/dm_verity.txt -o $out
    # prevent non-hermetic download of igvm-tooling / igvmgen
    mkdir -p msigvm-1.2.0
    ./igvm_builder.sh -d -k ${kata-kernel-uvm}/bzImage -v ${kata-image.verity}/dm_verity.txt -o $debug

    runHook postBuild
  '';

  meta = {
    description = "The Contrast runtime IGVM file defines the initial state of a pod-VM.";
    license = lib.licenses.asl20;
  };
}
