{ fetchFromGitHub
, rustPlatform
, openssl
, pkg-config
, libiconv
, zlib
, cmake
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  version = "0.6.2-5";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "genpolicy-${version}";
    hash = "sha256-R+kiyG3xLsoLBVTy1lmmqvDgoQuqfcV3DkfQtRCiYCw=";
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoHash = "sha256-MRVtChYQkiU92n/z+5r4ge58t9yVeOCdqs0zx81IQUY=";

  dontStrip = true;
  buildType = "debug";

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
