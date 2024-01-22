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
  version = "3.2.0-unstable-2024-01-18";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "069680738496d79a103965abcc1cc1fd91a8f24b";
    hash = "sha256-Ft2JgcGRwfGHqKebelkGgBBXRvvL/8ybMF6PqQMrk1c=";
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

  env.OPENSSL_NO_VENDOR = 1;

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
