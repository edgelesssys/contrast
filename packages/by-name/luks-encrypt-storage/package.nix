{
  stdenv,
  fetchFromGitHub,
  cryptsetup,
  e2fsprogs,
  mount,
  makeWrapper,
  lib,
}:

stdenv.mkDerivation rec {
  pname = "luks-encrypt-storage";
  version = "v0.13.0";

  src = fetchFromGitHub {
    owner = "confidential-containers";
    repo = "guest-components";
    rev = version;
    hash = "sha256-Bp8Ny9wqS2iDqZCiW2DUkgTGq3h1DJ92CZT9LCZx/h0=";
  };

  runtimeInputs = [
    cryptsetup
    e2fsprogs
    mount
  ];

  buildInputs = [ makeWrapper ];

  phases = [ "installPhase" ];

  installPhase = ''
    mkdir -p $out/bin

    # remove NixOS-incompatible shebang
    tail -n +2 $src/confidential-data-hub/hub/src/storage/scripts/luks-encrypt-storage > $out/bin/luks-encrypt-storage
    chmod +x $out/bin/luks-encrypt-storage

    wrapProgram $out/bin/luks-encrypt-storage --prefix PATH : ${
      lib.makeBinPath ([
        cryptsetup
        e2fsprogs
        mount
      ])
    }
  '';
}
