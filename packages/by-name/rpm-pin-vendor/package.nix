# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, dnf-plugins-core
, writeText
, writeTextDir
, writeShellApplication
, runCommand
, dnf4
, jq
, wget
, python3
, fakeroot
, nix
}:
let
  dnfConf = writeText "dnf.conf" ''
    [main]
    gpgcheck=True
    installonly_limit=3
    clean_requirements_on_remove=True
    skip_if_unavailable=True
    tsflags=nodocs
    pluginpath=${dnf-plugins-core}/${python3.sitePackages}/dnf-plugins
  '';
  reposdir = writeTextDir "yum.repos.d/cbl-mariner-2.repo" ''
    [cbl-mariner-2.0-prod-base-x86_64-yum]
    name=cbl-mariner-2.0-prod-base-x86_64-yum
    baseurl=https://packages.microsoft.com/yumrepos/cbl-mariner-2.0-prod-base-x86_64/
    repo_gpgcheck=0
    gpgcheck=1
    enabled=1
    gpgkey=https://packages.microsoft.com/yumrepos/cbl-mariner-2.0-prod-base-x86_64/repodata/repomd.xml.key
  '';
  # Contrast kata agent links with libseccomp from nix store
  # this packages only exists to satisfy the image builder
  packages = writeText "packages" ''
    kata-packages-uvm
    kata-packages-uvm-coco
    systemd
    libseccomp
  '';
  update_lockfile = writeShellApplication {
    name = "update_lockfile";
    runtimeInputs = [ dnf4 jq wget nix fakeroot ];
    text = builtins.readFile ./update_lockfile.sh;
  };
in
runCommand "rpm-pin-vendor" { meta.mainProgram = "update_lockfile"; } ''
  mkdir -p $out/bin
  cp ${lib.getExe update_lockfile} $out/bin/update_lockfile
  substituteInPlace $out/bin/update_lockfile \
    --replace-fail "@DNFCONFIG@" ${dnfConf} \
    --replace-fail "@REPOSDIR@" ${reposdir}/yum.repos.d \
    --replace-fail "@PACKAGESET@" ${packages}
''
