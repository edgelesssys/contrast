# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  fetchYarnDeps,
  stdenv,
  yarnConfigHook,
  yarnBuildHook,
  nodejs,
  contrast,

  # Configure the base URL when deploying previews under a subpath
  docusaurusBaseUrl ? "",
}:

stdenv.mkDerivation (finalAttrs: {
  pname = "contrast-docs";
  inherit (contrast) version;

  src = ../../../../docs;

  yarnOfflineCache = fetchYarnDeps {
    inherit (finalAttrs) pname version;
    yarnLock = finalAttrs.src + "/yarn.lock";
    hash = "sha256-0nQ5cuifWGIBfzinjnIxyv6reuFoV04FVXB6YhHV448=";
  };

  nativeBuildInputs = [
    yarnConfigHook
    yarnBuildHook
    nodejs
  ];

  env.CI = "true";

  postPatch = lib.optionalString (docusaurusBaseUrl != "") ''
    sed -i "s|baseUrl: '/contrast/',|baseUrl: '${docusaurusBaseUrl}',|" docusaurus.config.js
  '';

  installPhase = ''
    mkdir -p $out
    cp -R build/* $out
  '';
})
