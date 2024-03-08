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
  version = "0.6.2-5";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    # Latest released version of genpolicy
    # is too old for the path handling patch.
    # Using a commit from main for now.
    # rev = "genpolicy-${version}";
    rev = "401db3a3e75c699422537551e7862cd510fb68b0";
    hash = "sha256-dyYGGQPGWe6oVcAa48Kr/SsdSpUhwQZrRQ2d54BIac8=";
  };

  patches = [
    # TODO(malt3): drop this patch when msft fork adopted this from upstream
    (fetchpatch {
      name = "genpolicy_path_handling.patch";
      url = "https://github.com/kata-containers/kata-containers/commit/befef119ff4df2868cdc88d4273c8be965387793.patch";
      sha256 = "sha256-4pfYrP9KaPVcrFbm6DkiZUNckUq0fKWZPfCONW8/kso=";
    })
    # TODO(3u13r): drop this patch when msft fork adopted this from upstream
    (fetchpatch {
      name = "genpolicy_msft_settings_dev.patch";
      url = "https://github.com/kata-containers/kata-containers/commit/5398b6466c58676db7f73370e1a56f4fbb35d8cf.patch";
      sha256 = "sha256-cJ/uUF2F//QAP79AXu3tgfcByrWy2bvUjPfOAIZrtD8=";
    })
  ];

  patchFlags = [ "-p4" ];

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoHash = "sha256-WRSDqrOgSZVcJGN7PuyIqqmOSbrob75QNE2Ztb1L9Ww=";

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
