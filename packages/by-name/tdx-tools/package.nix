# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  python3Packages,
  fetchFromGitHub,
}:

python3Packages.buildPythonApplication rec {
  pname = "tdx-tools";
  version = "3.1";
  pyproject = true;

  src = fetchFromGitHub {
    owner = "canonical";
    repo = "tdx";
    tag = version;
    hash = "sha256-EfVZFsdVSpfi1jpmS/z31OKihhk4kVl4RxohYxBrYLA=";
  };

  build-system = [ python3Packages.setuptools ];

  dependencies = with python3Packages; [
    py-cpuinfo
  ];

  preBuild = ''
    cd tests/lib/tdx-tools
  '';

  meta = {
    homepage = "https://github.com/canonical/tdx";
    license = with lib.licenses; [
      gpl3Only
      asl20
    ];
  };
}
