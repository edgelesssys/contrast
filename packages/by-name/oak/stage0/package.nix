{
  lib,
  stdenv,
  rustPlatform,
  fetchFromGitHub,
  buildBazelPackage,
  bazel_7,
}:

buildBazelPackage rec {
  pname = "oak-stage0";
  version = "unstable-2025-12-17";
  bazel = bazel_7;

  src = fetchFromGitHub {
    owner = "project-oak";
    repo = "oak";
    rev = "bf836f091060ab43c263b87f8ef961871ae24117";
    hash = "sha256-bFLZMEfktlVMkuF/xvNW//OzA8RkK9UC+wweDTzDhjI=";
  };

  fetchAttrs = {
    hash =
      {
        x86_64-linux = "";
      }
      .${stdenv.hostPlatform.system} or (throw "No has for system: ${stdenv.hostPlatform.system}");
  };

  bazelTargets = [ "stage0_bin" ];

  bazelBuildFlags = [
    "--config=release"
  ];

  buildAttrs = {

  };

  meta = {
    description = "Meaningful control of data in distributed systems";
    homepage = "https://github.com/project-oak/oak";
    license = lib.licenses.asl20;
    mainProgram = "stage0";
    platforms = lib.platforms.all;
  };
}
