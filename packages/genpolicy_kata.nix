{ fetchFromGitHub
, rustPlatform
, openssl
, pkg-config
, protobuf
, libiconv
, zlib
, cmake
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  version = "0.6.2-1";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "4e2fce2d03dd7ed6101ba02e397374a0594fdd6d"; # danmihai1/genpolicy-main
    hash = "sha256-ydJrOu26rtiXKuDzccTgW6RhqUOzXVEm3840nsgI23Q=";
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoLock = {
    lockFile = "${src}/src/tools/genpolicy/Cargo.lock";
    outputHashes = {
      "tarfs-defs-0.1.0" = "sha256-J79fMuKOIVHEk6WvkLeM9IY5XQHyUJQOrwwMLvRvE60=";
    };
  };

  dontStrip = true;
  buildType = "debug";

  OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    cmake
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    libiconv
    zlib
  ];

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';
}
