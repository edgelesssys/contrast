# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, unzip
, fetchurl
, runCommand
, installShellFiles
}:

let
  json = builtins.fromJSON (builtins.readFile ./contrast-releases.json);
  listOrEmpty = list: field: if builtins.hasAttr field json then list.${field} else [ ];
  findVersion = field: version: lib.lists.findFirst (obj: obj.version == version) { hash = "unknown"; } (listOrEmpty json field);

  buildContrastRelease = { version, hash }: {
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
        };

        runtime = fetchurl {
          inherit version;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime.yml";
          inherit (findVersion "runtime.yml" version) hash;
          # runtime.yml was introduced in release v0.6.0
          passthru.exists = (builtins.compareVersions "v0.6.0" version) <= 0;
        };

        emojivoto-zip = fetchurl {
          # fetchurl instead of fetchzip since the hashes in contrast-release.json are computed from the zip file
          inherit version;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
          inherit (findVersion "emojivoto-demo.zip" version) hash;
          # emojivoto-demo.zip was introduced in version v0.5.0
          passthru.exists = (builtins.compareVersions "v0.5.0" version) <= 0 && (builtins.compareVersions version "v0.8.0") < 0;
        };

        emojivoto = fetchurl {
          inherit version;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.yml";
          inherit (findVersion "emojivoto-demo.yml" version) hash;
          # emojivoto-demo.yml was changed from zip to yml in version v0.8.0
          passthru.exists = (builtins.compareVersions "v0.8.0" version) <= 0;
        };
      in
      runCommand version
        {
          buildInputs = [ unzip installShellFiles ]; # needed to unzip emojivoto-demo.zip
        }
        (''
          mkdir -p $out/bin
          install -m 777 ${cli} $out/bin/contrast
          installShellCompletion --cmd contrast \
            --bash <($out/bin/contrast completion bash) \
            --fish <($out/bin/contrast completion fish) \
            --zsh <($out/bin/contrast completion zsh)

          install -m 644 ${coordinator} $out/coordinator.yml
        '' + lib.optionalString runtime.exists ''
          install -m 644 ${runtime} $out/runtime.yml
        '' + lib.optionalString emojivoto-zip.exists ''
          unzip ${emojivoto-zip} -d $out
        '' + lib.optionalString emojivoto.exists ''
          mkdir -p $out/deployment
          install -m 644 ${emojivoto} $out/deployment/emojivoto-demo.yml
        '');
  };
  releases = builtins.listToAttrs (builtins.map buildContrastRelease json.contrast);
  latestVersion = builtins.replaceStrings [ "." ] [ "-" ] (lib.last json.contrast).version;
in
releases // { latest = releases.${latestVersion}; }
