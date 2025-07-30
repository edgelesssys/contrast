{
  lib,
  llvmPackages_12,
  openssl_1_1,
  fetchFromGitHub,
  fetchzip,
  cmake,
}:

let
  intel-cryptographic-primitives = fetchzip {
    url = "https://github.com/intel/cryptography-primitives/archive/refs/tags/ippcp_2021.10.0.tar.gz";
    hash = "sha256-DfXsJ+4XqyjCD+79LUD53Cx8D46o1a4fAZa2UxGI1Xg=";
  };
in

llvmPackages_12.stdenv.mkDerivation (finalAttrs: {
  pname = "tdx-module";
  version = "1.5.06";

  src = fetchFromGitHub {
    owner = "intel";
    repo = "tdx-module";
    rev = "TDX_${finalAttrs.version}";
    hash = "sha256-TID+Z+K0HaCiFPY/PPFlMAZrgNDpzMiFYllWWJI4L3Y=";
  };

  postPatch = ''
    mkdir -p lib/ipp/ipp-crypto-ipp-crypto_2021_10_0
    cp -R ${intel-cryptographic-primitives}/* lib/ipp/ipp-crypto-ipp-crypto_2021_10_0/
    chmod -R u+rw lib/ipp/ipp-crypto-ipp-crypto_2021_10_0
  '';

  dontUseCmakeConfigure = true;

  nativeBuildInputs = [
    cmake
  ];

  buildInputs = [
    openssl_1_1
  ];

  buildPhase = ''
    make -d RELEASE=1 TDX_MODULE_BUILD_DATE=20240407 TDX_MODULE_BUILD_NUM=744 TDX_MODULE_UPDATE_VER=6
  '';

  meta = {
    description = "Trust Domain Extensions (TDX) is introducing new, architectural elements to help deploy hardware-isolated, virtual machines (VMs) called trust domains (TDs). Intel TDX is designed to isolate VMs from the virtual-machine manager (VMM)/hypervisor and any other non-TD software on the platform to protect TDs from a broad range of software";
    homepage = "https://github.com/intel/tdx-module";
    license = lib.licenses.mit;
    maintainers = with lib.maintainers; [ ];
    mainProgram = "tdx-module";
    platforms = lib.platforms.all;
  };
})
