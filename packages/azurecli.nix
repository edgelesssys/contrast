{ fetchurl
, python3
, stdenv
, azure-cli
, symlinkJoin
}:
let
  aks-preview = python3.pkgs.buildPythonPackage rec {
    pname = "aks-preview";
    version = "0.5.173";
    format = "wheel";
    src = fetchurl {
      url = "https://azcliprod.blob.core.windows.net/cli-extensions/aks_preview-0.5.173-py2.py3-none-any.whl";
      hash = "sha256-6BWX0CzL0oVrf9ljHjQU1jvmQiHXHGDcbhVIyVSH1u4=";
    };
    postInstall = ''
      ln -s $out/${python3.sitePackages} $out/${pname}
    '';
  };

  cliextensions = symlinkJoin {
    name = "cliextensions";
    paths = [ aks-preview ];
  };
in
azure-cli.overrideAttrs
  (oldAttrs: {
    postFixup = ''
      wrapProgram $out/bin/az \
        --set PYTHONPATH $PYTHONPATH \
        --set AZURE_EXTENSION_DIR ${cliextensions}
    '';
  })
