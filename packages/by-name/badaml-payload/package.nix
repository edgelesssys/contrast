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

    cp ./*.asl $out/

    for asl in $out/*.asl; do
      iasl -tc "$asl"
    done
  '';
}
