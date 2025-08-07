# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  unzip,
  fetchurl,
  runCommand,
  installShellFiles,
}:

let
  json = builtins.fromJSON (builtins.readFile ./contrast-releases.json);
  listOrEmpty = list: field: if builtins.hasAttr field json then list.${field} else [ ];
  findVersion =
    field: version:
    lib.lists.findFirst (obj: obj.version == version) { hash = "unknown"; } (listOrEmpty json field);

  versionLessThan = version: other: (builtins.compareVersions version other) < 0;
  versionGreaterEqual = version: other: (builtins.compareVersions version other) >= 0;

  allPlatforms = [
    "aks-clh-snp"
    "metal-qemu-snp"
    "metal-qemu-snp-gpu"
    "metal-qemu-tdx"
    "k3s-qemu-tdx"
    "k3s-qemu-snp"
    "k3s-qemu-snp-gpu"
    "rke2-qemu-tdx"
  ];

  forPlatforms =
    platforms: f:
    builtins.listToAttrs (
      lib.lists.map (platform: lib.attrsets.nameValuePair platform (f platform)) platforms
    );

  buildContrastRelease =
    { version, hash }:
    {
      name = builtins.replaceStrings [ "." ] [ "-" ] version;
      value =
        let
          cli = fetchurl {
            inherit hash version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/contrast";
          };

          coordinator = fetchurl {
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/coordinator.yml";
            inherit (findVersion "coordinator.yml" version) hash;
            passthru.exists =
              # coordinator.yml was replaced with a platform-specific version in release v0.10.0
              (versionLessThan version "v0.10.0")
              # coordinator.yml was re-introduced in release v1.6.0, ending the use of platform-specific Coordinator files
              && (versionGreaterEqual version "v1.6.0");
          };

          runtime = fetchurl {
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime.yml";
            inherit (findVersion "runtime.yml" version) hash;
            passthru.exists =
              # runtime.yml was introduced in release v0.6.0
              (versionGreaterEqual version "v0.6.0")
              # runtime.yml was replaced with a platform-specific version in release v1.1.0
              && (versionLessThan version "v1.0.0");
          };

          emojivoto-zip = fetchurl {
            # fetchurl instead of fetchzip since the hashes in contrast-release.json are computed from the zip file
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
            inherit (findVersion "emojivoto-demo.zip" version) hash;
            passthru.exists =
              # emojivoto-demo.zip was introduced in version v0.5.0
              (versionGreaterEqual version "v0.5.0")
              # emojivoto-demo.zip was replaced with the unzipped version in version v0.8.0
              && (versionLessThan version "v0.8.0");
          };

          emojivoto = fetchurl {
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.yml";
            inherit (findVersion "emojivoto-demo.yml" version) hash;
            # emojivoto-demo.yml was changed from zip to yml in version v0.8.0
            passthru.exists = versionGreaterEqual version "v0.8.0";
          };

          mysql-demo = fetchurl {
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/mysql-demo.yml";
            inherit (findVersion "mysql-demo.yml" version) hash;
            # mysql-demo.yml was introduced in version v1.2.0
            passthru.exists = versionGreaterEqual version "v1.2.0";
          };

          vault-demo = fetchurl {
            inherit version;
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/vault-demo.yml";
            inherit (findVersion "vault-demo.yml" version) hash;
            # vault-demo.yml was introduced in version v1.10.0
            passthru.exists = versionGreaterEqual version "v1.10.0";
          };

          coordinator-per-platform = forPlatforms allPlatforms (
            platform:
            fetchurl {
              inherit version;
              url = "https://github.com/edgelesssys/contrast/releases/download/${version}/coordinator-${platform}.yml";
              inherit (findVersion "coordinator-${platform}.yml" version) hash;
              passthru.exists =
                # the start date for this resource depends on the the version the platform was introduced
                (
                  if (platform == "metal-qemu-tdx" || platform == "metal-qemu-snp") then
                    (versionGreaterEqual version "v1.2.1")
                  else if (platform == "metal-qemu-snp-gpu" || platform == "k3s-qemu-snp-gpu") then
                    (versionGreaterEqual version "v1.4.0")
                  else
                    (versionGreaterEqual version "v1.1.0")
                )
                # platform-specific Coordinator files were removed in release v1.6.0
                && (versionLessThan version "v1.6.0");
            }
          );

          runtime-per-platform = forPlatforms allPlatforms (
            platform:
            fetchurl {
              inherit version;
              url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime-${platform}.yml";
              inherit (findVersion "runtime-${platform}.yml" version) hash;
              passthru.exists =
                if (platform == "metal-qemu-tdx" || platform == "metal-qemu-snp") then
                  (versionGreaterEqual version "v1.2.1")
                else if (platform == "metal-qemu-snp-gpu") then
                  (versionGreaterEqual version "v1.4.0")
                else if (platform == "k3s-qemu-snp-gpu") then
                  (versionGreaterEqual version "v1.4.0")
                  # platform removed in release v1.12.0
                  && (versionLessThan version "v1.12.0")
                else if (lib.hasPrefix "k3s-qemu" platform) then
                  (versionGreaterEqual version "v1.1.0")
                  # platform removed in release v1.12.0
                  && (versionLessThan version "v1.12.0")
                else
                  (versionGreaterEqual version "v1.1.0");
            }
          );

        in
        runCommand version
          {
            buildInputs = [
              unzip
              installShellFiles
            ];
          }
          (
            ''
              mkdir -p $out/bin
              install -m 777 ${cli} $out/bin/contrast
              installShellCompletion --cmd contrast \
                --bash <($out/bin/contrast completion bash) \
                --fish <($out/bin/contrast completion fish) \
                --zsh <($out/bin/contrast completion zsh)
            ''
            + lib.optionalString coordinator.exists ''
              install -m 644 ${coordinator} $out/coordinator.yml
            ''
            + lib.optionalString runtime.exists ''
              install -m 644 ${runtime} $out/runtime.yml
            ''
            + lib.optionalString emojivoto-zip.exists ''
              unzip ${emojivoto-zip} -d $out
            ''
            + lib.optionalString emojivoto.exists ''
              mkdir -p $out/deployment
              install -m 644 ${emojivoto} $out/deployment/emojivoto-demo.yml
            ''
            + lib.optionalString mysql-demo.exists ''
              mkdir -p $out/deployment
              install -m 644 ${mysql-demo} $out/deployment/mysql-demo.yml
            ''
            + lib.optionalString vault-demo.exists ''
              mkdir -p $out/deployment
              install -m 644 ${vault-demo} $out/deployment/vault-demo.yml
            ''
            + lib.concatStrings (
              lib.attrsets.mapAttrsToList (
                platform: file:
                lib.optionalString file.exists ''
                  install -m 644 ${file} $out/coordinator-${platform}.yml
                ''
              ) coordinator-per-platform
            )
            + lib.concatStrings (
              lib.attrsets.mapAttrsToList (
                platform: file:
                lib.optionalString file.exists ''
                  install -m 644 ${file} $out/runtime-${platform}.yml
                ''
              ) runtime-per-platform
            )
          );
    };
  releases = builtins.listToAttrs (builtins.map buildContrastRelease json.contrast);
  latestVersion = builtins.replaceStrings [ "." ] [ "-" ] (lib.last json.contrast).version;
in
releases // { latest = releases.${latestVersion}; }
