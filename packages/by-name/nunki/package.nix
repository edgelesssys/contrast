{ lib
, buildGoModule
, genpolicy-msft
, genpolicy ? genpolicy-msft
}:

buildGoModule rec {
  pname = "nunki";
  version = builtins.readFile ../../../version.txt;

  outputs = subPackages ++ [ "out" ];

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-W6mphSRaIIGE9hp0Ga0+7syj1HkFaCKJykxfO2PbB9I=";

  subPackages = [ "coordinator" "initializer" "cli" ];

  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/assets/genpolicy
    install -D ${genpolicy.settings-dev}/genpolicy-settings.json cli/assets/genpolicy-settings.json
    install -D ${genpolicy.rules}/genpolicy-rules.rego cli/assets/genpolicy-rules.rego
  '';

  CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X main.version=v${version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  postInstall = ''
    for sub in ${builtins.concatStringsSep " " subPackages}; do
      mkdir -p "''${!sub}/bin"
      mv "$out/bin/$sub" "''${!sub}/bin/$sub"
    done

    # ensure no binary is left in out
    rmdir "$out/bin/"

    # rename the cli binary to nunki
    mv "$cli/bin/cli" "$cli/bin/nunki"
  '';
  meta.mainProgram = "nunki";
}
