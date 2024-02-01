{ lib
, fetchurl
, fetchFromGitHub
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

  passthru = rec {
    settings = fetchurl {
      name = "${pname}-${version}-settings";
      url = "https://raw.githubusercontent.com/kata-containers/kata-containers/${src.rev}/src/tools/genpolicy/genpolicy-settings.json";
      hash = "sha256-6SbX/dyi9OIHH03TBFBfu5BJ921fNhClrPLfqMyX3hQ=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-settings.json";
    };

    rules = fetchurl {
      name = "${pname}-${version}-rules";
      url = "https://raw.githubusercontent.com/kata-containers/kata-containers/${src.rev}/src/tools/genpolicy/rules.rego";
      hash = "sha256-Dru5UPWlJM3TEmMUpG+rMKbrJmAb3/v3vlUOZZN3IPI=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-rules.rego";
    };
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "genpolicy";
    license = lib.licenses.asl20;
  };
}
