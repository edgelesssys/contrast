# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  fetchurl,
  linuxManualConfig,
  stdenvNoCC,
  fetchzip,
}:

let
  configfile = stdenvNoCC.mkDerivation rec {
    pname = "kata-kernel-config-confidential";
    version = "3.7.0";

    src = fetchzip {
      url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
      hash = "sha256-SY75Ond2WLkY17Zal22GXgNKB3L1LGIyLKv8H/M0Wbw=";
    };

    # We don't use an initrd.
    postPatch = ''
      config=$(find . -regex '.*/config-[0-9.-]+-confidential')
      substituteInPlace $config \
        --replace-fail 'CONFIG_INITRAMFS_SOURCE="initramfs.cpio.gz"' 'CONFIG_INITRAMFS_SOURCE=""'
    '';

    dontBuild = true;

    installPhase = ''
      runHook preInstall

      cp $config $out

      runHook postInstall
    '';
  };
in

linuxManualConfig rec {
  version = "6.7";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    hash = "sha256-7zEUSiV20IDYwxaY6D7J9mv5fGd/oqrw1bu58zRbEGk=";
  };

  inherit configfile;
  allowImportFromDerivation = true;
}
