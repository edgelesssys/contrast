{ lib
, fetchFromGitHub
, fetchurl
, fetchpatch
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
  version = "3.2.0.azl0.genpolicy1";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "refs/tags/${version}";
    hash = "sha256-A+xlX7OsGUiYnvceUO5DKz14ASqbJnAg0SVJY5laxqs=";
  };

  patches = [
    # TODO(malt3): drop this patch when msft fork adopted this from upstream
    (fetchpatch {
      name = "genpolicy_path_handling.patch";
      url = "https://github.com/kata-containers/kata-containers/commit/befef119ff4df2868cdc88d4273c8be965387793.patch";
      sha256 = "sha256-4pfYrP9KaPVcrFbm6DkiZUNckUq0fKWZPfCONW8/kso=";
    })
  ];

  patchFlags = [ "-p4" ];

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoHash = "sha256-BWFD3gm1big/frVCWksLyTcn3R/QSefavQ2g1EPsBkU=";

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
      url = "https://github.com/microsoft/kata-containers/releases/download/${version}/genpolicy-settings.json";
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
      url = "https://github.com/microsoft/kata-containers/releases/download/${version}/rules.rego";
      hash = "sha256-piPyARaIwtJ5CZiVlQ+t793Z80IIpWFG8iN7jHgBe6E=";
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
