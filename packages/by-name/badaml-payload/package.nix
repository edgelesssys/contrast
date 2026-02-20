# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  stdenvNoCC,
  acpica-tools,
}:

stdenvNoCC.mkDerivation {
  name = "badaml-payload";
  src = ./../../../tools/badaml-payload;
  preferLocalBuild = true;

  nativeBuildInputs = [ acpica-tools ];

  buildPhase = ''
    mkdir -p $out

    INITRD_START="$(cat ./initrd_start)"

    for tmpl in *.asl.tmpl; do
      outFile="''${tmpl%.tmpl}"
      substitute "$tmpl" "$outFile" --subst-var INITRD_START
    done

    cp ./*.asl $out/

    for asl in $out/*.asl; do
      iasl -tc "$asl"
    done
  '';
}
