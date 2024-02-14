{ lib
, fetchFromGitHub
, fetchurl
, applyPatches
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

  passthru = rec {
    settings = fetchurl {
      name = "${pname}-${version}-settings";
      url = "https://github.com/microsoft/kata-containers/releases/download/genpolicy-${version}/genpolicy-settings.json";
      hash = "sha256-Q19H7Oj8c7SlPyib96fSRZhx/nJ96HXb8dfb9Y/Rsw8=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-settings.json";
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings;
      patches = [ ./genpolicy_msft_settings_dev.patch ];
    };

    rules = fetchurl {
      name = "${pname}-${version}-rules";
      url = "https://github.com/microsoft/kata-containers/releases/download/genpolicy-${version}/rules.rego";
      hash = "sha256-D58bmeOu9MMBCaNoF4mmoG6rzVKRvCesZxOFkBdvxd8=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-rules.rego";
    };

    rules-coordinator = applyPatches {
      src = rules;
      patches = [ ./genpolicy_msft_rules_coordinator.patch ];
    };
  };

  meta = {
    changelog = "https://github.com/microsoft/kata-containers/releases/tag/genpolicy-${version}";
    homepage = "https://github.com/microsoft/kata-containers";
    mainProgram = "genpolicy";
    licesnse = lib.licenses.asl20;
  };
}
