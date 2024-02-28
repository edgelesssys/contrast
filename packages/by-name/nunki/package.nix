{ lib
, buildGoModule
, buildGoTest
, genpolicy-msft
, genpolicy ? genpolicy-msft
, nunki
}:
let
  e2e = buildGoTest rec {
    inherit (nunki) version src proxyVendor vendorHash prePatch CGO_ENABLED;
    pname = "${nunki.pname}-e2e";

    tags = [ "e2e" ];

    ldflags = [ "-s" ];

    subPackages = [ "e2e/openssl" ];
  };
in

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
        (lib.fileset.difference
          (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) root)
          (path.append root "service-mesh"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-VBCTnRx4BBvG/yedChE55ZQbsaFk2zDcXtXof9v3XNI=";

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

  passthru.e2e = e2e;

  meta.mainProgram = "nunki";
}
