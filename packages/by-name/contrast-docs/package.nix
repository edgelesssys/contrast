{ lib
, fetchYarnDeps
, mkYarnPackage
, contrast

  # Configure the base URL when deploying previews under a subpath
, docusaurusBaseUrl ? ""
}:

mkYarnPackage rec {
  pname = "contrast-docs";
  inherit (contrast) version;

  src = ../../../docs;

  packageJSON = "${src}/package.json";
  offlineCache = fetchYarnDeps {
    yarnLock = "${src}/yarn.lock ";
    hash = "sha256-10IMmokzzQb0i3VzODj8t3V75XEG7HmkbVGO4NyAS0I=";
  };

  configurePhase = ''
    cp -r $node_modules node_modules
    chmod +w node_modules
  '' + lib.optionalString (docusaurusBaseUrl != "") ''
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
