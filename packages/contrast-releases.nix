{pkgs, fetchurl, lib, runCommand}:

let
  json = builtins.fromJSON (builtins.readFile ./contrast-releases.json);
  findVersion = list: version: lib.lists.findFirst (obj: obj.version == version) (throw "version ${version} not found") list;

  buildContrastRelease = {
    version, hash 
  }: {
    name = builtins.replaceStrings ["."] ["-"] version;
    value = 
    let
      cli = fetchurl {
        inherit hash version;
        url = "https://github.com/edgelesssys/contrast/releases/download/${version}/contrast";
      };

      coordinator = 
        let
          coordinatorRelease = findVersion json."coordinator.yml" version;
        in fetchurl {
          inherit (coordinatorRelease) version hash;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/coordinator.yml";
        };

      # runtime.yml was introduced in release v0.6.0
      runtimeExists = (builtins.compareVersions "v0.6.0" version) <= 0;
      runtime = 
        let
          runtimeRelease = findVersion json."runtime.yml" version;
        in 
        fetchurl {
          inherit (runtimeRelease) version hash;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime.yml";
        };

      # emojivoto-demo.zip was introduced in version v0.5.0
      emojivotoExists = (builtins.compareVersions "v0.5.0" version) <= 0;
      emojivoto =
        let
          emojivotoRelease = findVersion json."emojivoto-demo.zip" version;
        in
        # fetchurl instead of fetchzip since the hashes in contrast-release.json are computed from the zip file
        fetchurl {
          inherit (emojivotoRelease) version hash;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
        };
    in
    runCommand version {
      buildInputs = [ pkgs.unzip ]; # needed to unzip emojivoto-demo.zip
    } 
    (''
      mkdir -p $out/bin
      install -m 777 ${cli} $out/bin/contrast
      install -m 644 ${coordinator} $out/coordinator.yml
    '' + lib.optionalString runtimeExists ''
    install -m 644 ${runtime} $out/runtime.yml
    '' + lib.optionalString emojivotoExists ''
    mkdir -p $out/deployment
    unzip ${emojivoto} -d $out/deployment
    '');
  };
in
builtins.listToAttrs (builtins.map buildContrastRelease json.contrast)
