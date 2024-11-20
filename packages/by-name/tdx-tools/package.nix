# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  python3Packages,
  fetchFromGitHub,
}:

python3Packages.buildPythonApplication rec {
  pname = "tdx-tools";
  version = "noble-24.04";
  pyproject = true;

  src = fetchFromGitHub {
    owner = "canonical";
    repo = "tdx";
    rev = version;
    sha256 = "sha256-4Uzsnrf/B3awMutSPSF9PeOZ68mstNzQXnaD11nHWD4=";
  };

  build-system = [ python3Packages.setuptools ];

  dependencies = with python3Packages; [
    py-cpuinfo
  ];

  preBuild = ''
    cd tests/lib/tdx-tools
  '';
}
