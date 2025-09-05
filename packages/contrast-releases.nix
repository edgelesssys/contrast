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

  findInstall =
    filename:
    {
      "contrast" = path: ''
        mkdir -p $out/bin
        install -m 777 ${path} $out/bin/contrast
        installShellCompletion --cmd contrast \
          --bash <($out/bin/contrast completion bash) \
          --fish <($out/bin/contrast completion fish) \
          --zsh <($out/bin/contrast completion zsh)
      '';
      "emojivoto-demo.zip" = path: ''
        unzip ${path} -d $out
      '';
      "emojivoto-demo.yml" = path: ''
        mkdir -p $out/deployment
        install -m 644 ${path} $out/deployment/emojivoto-demo.yml
      '';
      "mysql-demo.yml" = path: ''
        mkdir -p $out/deployment
        install -m 644 ${path} $out/deployment/mysql-demo.yml
      '';
      "vault-demo.yml" = path: ''
        mkdir -p $out/deployment
        install -m 644 ${path} $out/deployment/vault-demo.yml
      '';
    }
    ."${filename}" or
    # default to generic install for other files.
    (path: "install -m 644 ${path} $out/${filename}");

  findVersion = versions: version: lib.lists.findFirst (f: f.version == version) { } versions;

  fetchReleasedFile =
    {
      filename,
      version,
      hash,
      ...
    }:
    fetchurl {
      url = "https://github.com/edgelesssys/contrast/releases/download/${version}/${filename}";
      inherit hash;
    };

  buildContrastRelease =
    { version, ... }:
    let
      files = lib.pipe json [
        (lib.filterAttrs (_: versions: lib.any (v: v.version == version) versions))
        (lib.mapAttrsToList (filename: versions: { inherit filename; } // (findVersion versions version)))
        (lib.map (file: file // { path = fetchReleasedFile file; }))
        (lib.map (file: file // { install = findInstall file.filename; }))
      ];
      installFiles = lib.concatStringsSep "\n" (lib.map (file: file.install file.path) files);
    in
    {
      name = builtins.replaceStrings [ "." ] [ "-" ] version;
      value = runCommand version {
        buildInputs = [
          unzip
          installShellFiles
        ];
      } installFiles;
    };

  releases = builtins.listToAttrs (builtins.map buildContrastRelease json.contrast);
  latestVersion = builtins.replaceStrings [ "." ] [ "-" ] (lib.last json.contrast).version;
in
releases // { latest = releases.${latestVersion}; }
