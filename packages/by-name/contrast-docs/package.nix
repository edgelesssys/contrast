# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  fetchYarnDeps,
  mkYarnPackage,
  contrast,

  # Configure the base URL when deploying previews under a subpath
  docusaurusBaseUrl ? "",
}:

mkYarnPackage rec {
  pname = "contrast-docs";
  inherit (contrast) version;

  src = ../../../docs;

  packageJSON = "${src}/package.json";
  offlineCache = fetchYarnDeps {
    yarnLock = "${src}/yarn.lock ";
    hash = "sha256-ZsTGaObMXLeVPgU61WJbg6Q8Vje5PKNJEi5Ic1SNiLQ=";
  };

  configurePhase = ''
    cp -r $node_modules node_modules
    chmod +w node_modules
  ''
  + lib.optionalString (docusaurusBaseUrl != "") ''
    sed -i "s|baseUrl: '/contrast/',|baseUrl: '${docusaurusBaseUrl}',|" docusaurus.config.js
  '';

  buildPhase = ''
    export HOME=$(mktemp -d)
    yarn --offline build
  '';

  distPhase = "true";

  installPhase = ''
    mkdir -p $out
    cp -R build/* $out
  '';
}
