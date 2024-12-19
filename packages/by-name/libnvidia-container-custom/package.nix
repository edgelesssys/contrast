# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# Upstream package from https://github.com/NixOS/nixpkgs/blob/nixos-24.11/pkgs/by-name/li/libnvidia-container/package.nix#L145
# Adapted to use custom paths for binary resolving specialized to the NixOS image we use this in. As this is incompatible with
# non-NixOS deployments, this cannot be upstreamed.

{
  stdenv,
  lib,
  addDriverRunpath,
  fetchFromGitHub,
  pkg-config,
  elfutils,
  libcap,
  libseccomp,
  rpcsvc-proto,
  libtirpc,
  makeWrapper,
  substituteAll,
  removeReferencesTo,
  replaceVars,
  go,
  binaryPaths ? [
    "/run/current-system/sw"
    "/run/opengl-driver/lib"
  ],
}:
let
  modprobeVersion = "550.54.14";
  nvidia-modprobe = fetchFromGitHub {
    owner = "NVIDIA";
    repo = "nvidia-modprobe";
    rev = modprobeVersion;
    sha256 = "sha256-iBRMkvOXacs/llTtvc/ZC5i/q9gc8lMuUHxMbu8A+Kg=";
  };
  modprobePatch = substituteAll {
    src = ./modprobe.patch;
    inherit modprobeVersion;
  };
in
stdenv.mkDerivation rec {
  pname = "libnvidia-container-custom";
  version = "1.16.2";

  src = fetchFromGitHub {
    owner = "NVIDIA";
    repo = "libnvidia-container";
    rev = "v${version}";
    sha256 = "sha256-hX+2B+0kHiAC2lyo6kwe7DctPLJWgRdbhlc316OO3r8=";
  };

  patches = [
    (replaceVars ./fix-library-resolving.patch {
      inherit (addDriverRunpath) driverLink;
      binaryPath = lib.makeBinPath binaryPaths;
    })

    ./inline-c-struct.patch
  ];

  postPatch = ''
    sed -i \
      -e 's/^REVISION ?=.*/REVISION = ${src.rev}/' \
      -e 's/^COMPILER :=.*/COMPILER = $(CC)/' \
      mk/common.mk
    sed -i \
      -e 's/^GIT_TAG ?=.*/GIT_TAG = ${version}/' \
      -e 's/^GIT_COMMIT ?=.*/GIT_COMMIT = ${src.rev}/' \
      versions.mk
    mkdir -p deps/src/nvidia-modprobe-${modprobeVersion}
    cp -r ${nvidia-modprobe}/* deps/src/nvidia-modprobe-${modprobeVersion}
    chmod -R u+w deps/src
    pushd deps/src
    patch -p0 < ${modprobePatch}
    touch nvidia-modprobe-${modprobeVersion}/.download_stamp
    popd
    # 1. replace DESTDIR=$(DEPS_DIR) with empty strings to prevent copying
    #    things into deps/src/nix/store
    # 2. similarly, remove any paths prefixed with DEPS_DIR
    # 3. prevent building static libraries because we don't build static
    #    libtirpc (for now)
    # 4. prevent installation of static libraries because of step 3
    # 5. prevent installation of libnvidia-container-go.so twice
    sed -i Makefile \
      -e 's#DESTDIR=\$(DEPS_DIR)#DESTDIR=""#g' \
      -e 's#\$(DEPS_DIR)\$#\$#g' \
      -e 's#all: shared static tools#all: shared tools#g' \
      -e '/$(INSTALL) -m 644 $(LIB_STATIC) $(DESTDIR)$(libdir)/d' \
      -e '/$(INSTALL) -m 755 $(libdir)\/$(LIBGO_SHARED) $(DESTDIR)$(libdir)/d'
  '';

  enableParallelBuilding = true;

  preBuild = ''
    HOME="$(mktemp -d)"
  '';

  env.NIX_CFLAGS_COMPILE = toString [ "-I${lib.getInclude libtirpc}/include/tirpc" ];
  NIX_LDFLAGS = [
    "-L${lib.getLib libtirpc}/lib"
    "-ltirpc"
  ];

  nativeBuildInputs = [
    pkg-config
    go
    rpcsvc-proto
    makeWrapper
    removeReferencesTo
  ];

  buildInputs = [
    elfutils
    libcap
    libseccomp
    libtirpc
  ];

  makeFlags = [
    "WITH_LIBELF=yes"
    "prefix=$(out)"
    "CFLAGS=-DWITH_TIRPC"
  ];

  postInstall =
    let
      inherit (addDriverRunpath) driverLink;
      libraryPath = lib.makeLibraryPath [
        "$out"
        driverLink
        "${driverLink}-32"
      ];
    in
    ''
      remove-references-to -t "${go}" $out/lib/libnvidia-container-go.so.${version}
      wrapProgram $out/bin/nvidia-container-cli --prefix LD_LIBRARY_PATH : ${libraryPath}
    '';
  disallowedReferences = [ go ];
}
