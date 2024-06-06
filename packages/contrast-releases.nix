{fetchurl, runCommand}:

let
json = builtins.fromJSON (builtins.readFile ./contrast-releases.json);

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
  in
  runCommand version {} 
  ''
    mkdir -p $out/bin
    install -m 777 ${cli} $out/bin/contrast
  '';
};

in
builtins.listToAttrs (builtins.map buildContrastRelease json.contrast)
