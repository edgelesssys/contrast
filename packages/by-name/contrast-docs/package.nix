{ fetchYarnDeps
, mkYarnPackage
, contrast
}:

mkYarnPackage rec {
  pname = "contrast-docs";
  inherit (contrast) version;

  src = ../../../docs;

  packageJSON = "${src}/package.json";
  offlineCache = fetchYarnDeps {
    yarnLock = "${src}/yarn.lock ";
    hash = "sha256-8TkRMs8TpF53ehJ1WlXf/AHcGfgD7KCjbH6ZlZDKo0E=";
  };

  configurePhase = ''
    cp -r $node_modules node_modules
    chmod +w node_modules
  '';

  buildPhase = ''
    export HOME=$(mktemp -d)
    yarn --offline build
  '';

  distPhase = "true";

  installPhase = ''
    mkdir -p $out
    cp -R build/* $out
    cp CNAME $out
  '';
}
