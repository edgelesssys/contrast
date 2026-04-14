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
    key:
    {
      "contrast-x86_64-linux" = path: ''
        mkdir -p $out/bin
        install -m 777 ${path} $out/bin/contrast
        installShellCompletion --cmd contrast \
          --bash <($out/bin/contrast completion bash) \
          --fish <($out/bin/contrast completion fish) \
          --zsh <($out/bin/contrast completion zsh)
      '';
      "coordinator.yml" = path: ''
        mkdir -p $out/deployment
        install -m 644 ${path} $out/deployment/coordinator.yml
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
    ."${key}" or
    # default to generic install for other files.
    (path: "install -m 644 ${path} $out/${key}");

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
        (lib.mapAttrsToList (key: versions: { inherit key; } // (findVersion versions version)))
        # Default the URL filename to the JSON key. Historical entries set
        # `filenameOverride` when the artifact name on the release page differs
        # from the current naming convention.
        (lib.map (file: file // { filename = file.filenameOverride or file.key; }))
        (lib.map (file: file // { path = fetchReleasedFile file; }))
        (lib.map (file: file // { install = findInstall file.key; }))
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

  # The linux CLI is the canonical version index: it's the only artifact
  # shipped in every release from v0.2.0 onwards, so its entries enumerate
  # all known versions in order.
  releases = builtins.listToAttrs (map buildContrastRelease json."contrast-x86_64-linux");
  latestVersion =
    builtins.replaceStrings [ "." ] [ "-" ]
      (lib.last json."contrast-x86_64-linux").version;
in
releases // { latest = releases.${latestVersion}; }
