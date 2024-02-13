{ lib
, buildGoModule
}:

buildGoModule rec {
  pname = "service-mesh";
  version = builtins.readFile ../../../version.txt;

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../service-mesh;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-uNd8HOd1HXhTvksoVLFsoIof/3VNnBZDLeEraYs5i3s=";

  subPackages = [ "." ];

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

  meta.mainProgram = "service-mesh";
}
