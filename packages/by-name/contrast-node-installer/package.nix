{ lib
, buildGoModule
, contrast
}:

buildGoModule rec {
  pname = "contrast-node-installer";
  inherit (contrast) version;

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../node-installer;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (lib.fileset.fileFilter (file: lib.hasSuffix ".toml" file.name) root)
        (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-uCmDgm68Q9ClAqKYIHVYIDAVWkUm2tyQ4eoP3L8+wLg=";

  subPackages = [ "." ];

  CGO_ENABLED = 0;
  ldflags = [
    "-s"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  meta.mainProgram = "node-installer";
}
