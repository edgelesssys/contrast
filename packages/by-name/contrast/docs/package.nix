# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenv,
  yarn-berry_4,
  nodejs,
  contrast,

  # Configure the base URL when deploying previews under a subpath
  docusaurusBaseUrl ? "",
}:

stdenv.mkDerivation (finalAttrs: {
  pname = "contrast-docs";
  inherit (contrast) version;

  src = ../../../../docs;

  missingHashes = finalAttrs.src + "/missing-hashes.json";
  yarnOfflineCache = yarn-berry_4.fetchYarnBerryDeps {
    inherit (finalAttrs) src missingHashes;
    hash = "sha256-Cp91Dw2KxXyWT4641sw001tj9TygEvLPENwy7X7/E6o=";
  };

  nativeBuildInputs = [
    yarn-berry_4
    yarn-berry_4.yarnBerryConfigHook
    nodejs
  ];

  env.CI = "true";

  postPatch = lib.optionalString (docusaurusBaseUrl != "") ''
    sed -i "s|baseUrl: '/contrast/',|baseUrl: '${docusaurusBaseUrl}',|" docusaurus.config.js
  '';

  buildPhase = ''
    runHook preBuild
    yarn build
    runHook postBuild
  '';

  installPhase = ''
    mkdir -p $out
    cp -R build/* $out
  '';
})
