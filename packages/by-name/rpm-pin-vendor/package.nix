# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  dnf-plugins-core,
  writeText,
  writeTextDir,
  writeShellApplication,
  runCommand,
  dnf4,
  jq,
  wget,
  python3,
  fakeroot,
  nix,
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
  reposdir = writeTextDir "yum.repos.d/azurelinux-3.0.repo" ''
    [azurelinux-3.0-prod-base-x86_64]
    name=azurelinux-3.0-prod-base-x86_64
    baseurl=https://packages.microsoft.com/yumrepos/azurelinux-3.0-prod-base-x86_64/
    repo_gpgcheck=0
    gpgcheck=1
    enabled=1
    gpgkey=https://packages.microsoft.com/yumrepos/azurelinux-3.0-prod-base-x86_64/repodata/repomd.xml.key
  '';
  update_lockfile = writeShellApplication {
    name = "update_lockfile";
    runtimeInputs = [
      dnf4
      jq
      wget
      nix
      fakeroot
    ];
    text = builtins.readFile ./update_lockfile.sh;
  };
in

runCommand "rpm-pin-vendor" { meta.mainProgram = "update_lockfile"; } ''
  mkdir -p $out/bin
  cp ${lib.getExe update_lockfile} $out/bin/update_lockfile
  substituteInPlace $out/bin/update_lockfile \
    --replace-fail "@DNFCONFIG@" ${dnfConf} \
    --replace-fail "@REPOSDIR@" ${reposdir}/yum.repos.d
''
