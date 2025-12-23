# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenv,
  fetchFromGitHub,
  nodejs,
  pnpm_9,
  pnpmConfigHook,
  fetchPnpmDeps,
  makeWrapper,
}:

stdenv.mkDerivation (finalAttrs: {
  pname = "github-matrix-parser";
  version = "unstable-2025-12-22";

  src = fetchFromGitHub {
    owner = "katexochen";
    repo = "github-matrix-parser";
    rev = "37e5ad1a11714d4e9188bc75b712a458746af7b1";
    hash = "sha256-csdpCb3og7I5Tm+XZUMS9Gvbsac4l7H+tUcs5/inpPU=";
  };

  nativeBuildInputs = [
    nodejs
    pnpm_9
    pnpmConfigHook
    makeWrapper
  ];

  pnpmDeps = fetchPnpmDeps {
    inherit (finalAttrs) pname version src;
    pnpm = pnpm_9;
    fetcherVersion = 3;
    hash = "sha256-mQysvhSwfjrVSbUnCNFgW4k25YZmuFtxaKtYqP+17nk=";
  };

  dontBuild = true;

  installPhase = ''
    runHook preInstall

    mkdir -p $out/bin $out/lib
    cp -r . $out/lib/github-matrix-parser
    makeWrapper ${nodejs}/bin/node $out/bin/github-matrix-parser \
      --add-flags "$out/lib/github-matrix-parser/cli.js"

    runHook postInstall
  '';

  meta = {
    description = "Parse and lint GitHub actions matrix combinations";
    homepage = "https://github.com/katexochen/github-matrix-parser";
    license = lib.licenses.isc;
    mainProgram = "github-matrix-parser";
    platforms = lib.platforms.all;
  };
})
