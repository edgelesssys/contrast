{ lib
, fetchFromGitHub
, rustPlatform
, openssl
, pkg-config
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
    rev = "genpolicy-${version}";
    hash = "sha256-/BMaKa8btqaiumlCGkwn6sJ0nMzm8fTbOn/54B2VkuI=";
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoHash = "sha256-FHJY6kUVK9UDkDJj6l8SHfV3AwHOnqHK7cp09pU1DEA=";

  OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    pkg-config
    cmake
  ];

  buildInputs = [
    openssl
    openssl.dev
    libiconv
    zlib
  ];
}
