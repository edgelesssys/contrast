{ lib
, fetchFromGitHub
, rustPlatform
, cmake
}:

rustPlatform.buildRustPackage rec {
  pname = "igvmmeasure";
  version = "0.1.0-unstable-2024-03-18";

  src = fetchFromGitHub {
    owner = "coconut-svsm";
    repo = "svsm";
    # TODO(malt3): Use a released version once available.
    rev = "3fd89b35c56477729987390e35ad235102ccc48f";
    hash = "sha256-SuDb+S6F1S+8apyQI/t+U6L6dlYg+zGJI9s1HWglGm0=";
  };

  cargoBuildFlags = "-p igvmmeasure";

  cargoLock = {
    lockFile = "${src}/Cargo.lock";
    outputHashes = {
      "packit-0.1.1" = "sha256-BLVpKYjrqTwEAPgL7V1xwMnmNn4B8bA38GSmrry0GIM=";
    };
  };

  meta = {
    changelog = "https://github.com/coconut-svsm/svsm/releases/tag/${version}";
    homepage = "https://github.com/coconut-svsm/svsm";
    mainProgram = "igvmmeasure";
    license = lib.licenses.mit;
  };
}
