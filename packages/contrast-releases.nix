{ lib
, unzip
, fetchurl
, runCommand
}:

let
  json = builtins.fromJSON (builtins.readFile ./contrast-releases.json);
  findVersion = list: version: lib.lists.findFirst (obj: obj.version == version) { hash = "unknown"; } list;

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
          hash = (findVersion json."coordinator.yml" version).hash;
        };

        runtime = fetchurl {
          inherit version;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime.yml";
          hash = (findVersion json."runtime.yml" version).hash;
          # runtime.yml was introduced in release v0.6.0
          passthru.exists = (builtins.compareVersions "v0.6.0" version) <= 0;
        };

        emojivoto = fetchurl {
          # fetchurl instead of fetchzip since the hashes in contrast-release.json are computed from the zip file
          inherit version;
          url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
          hash = (findVersion json."emojivoto-demo.zip" version).hash;
          # emojivoto-demo.zip was introduced in version v0.5.0
          passthru.exists = (builtins.compareVersions "v0.5.0" version) <= 0;
        };
      in
      runCommand version
        {
          buildInputs = [ unzip ]; # needed to unzip emojivoto-demo.zip
        }
        (''
          mkdir -p $out/bin
          install -m 777 ${cli} $out/bin/contrast
          install -m 644 ${coordinator} $out/coordinator.yml
        '' + lib.optionalString runtime.exists ''
          install -m 644 ${runtime} $out/runtime.yml
        '' + lib.optionalString emojivoto.exists ''
          unzip ${emojivoto} -d $out
        '');
  };
  releases = builtins.listToAttrs (builtins.map buildContrastRelease json.contrast);
  latestVersion = builtins.replaceStrings [ "." ] [ "-" ] (lib.last json.contrast).version;
in
releases // {latest = releases.${latestVersion};}
