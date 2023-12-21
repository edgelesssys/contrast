{ pkgs
, goVendorHash
}:

with pkgs;

let
  # The source of our local Go module. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  goFiles = lib.fileset.unions [
    ../go.mod
    ../go.sum
    (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) ../.)
  ];

  # Builder function for Go packages of our local module.
  buildGoSubPackage = subpackage: attrs: pkgs.callPackage
    ({ buildGoModule }: buildGoModule ({
      name = subpackage;
      src = lib.fileset.toSource {
        root = ../.;
        fileset = goFiles;
      };
      subPackages = [ subpackage ];
      CGO_ENABLED = 0;
      ldflags = [ "-s" "-w" "-buildid=" ];
      proxyVendor = true;
      vendorHash = goVendorHash;
      checkPhase = ''
        runHook preCheck

        export GOFLAGS=''${GOFLAGS//-trimpath/}
        buildGoDir test ./...

        runHook postCheck
      '';
      meta.mainProgram = "${subpackage}";
    } // attrs))
    { };

  buildContainer = drv: pkgs.dockerTools.buildImage {
    inherit (drv) name;
    tag = "latest";
    copyToRoot = with pkgs.dockerTools; [
      caCertificates
    ];
    config = {
      Cmd = [ "${lib.getExe drv}" ];
    };
  };

  pushContainer = container: pkgs.writeShellApplication {
    name = "push";
    runtimeInputs = with pkgs; [ crane gzip ];
    text = ''
      imageName="$1"
      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT
      gunzip < "${container}" > "$tmpdir/image.tar"
      crane push "$tmpdir/image.tar" "$imageName"
    '';
  };
in
rec {
  coordinator = buildContainer (buildGoSubPackage "coordinator" { });
  initializer = buildContainer (buildGoSubPackage "initializer" { });
  cli = buildGoSubPackage "cli" {
    postPatch = ''
      echo subsituting genpolicyPath
      substituteInPlace cli/runtime.go \
        --replace 'genpolicyPath = "genpolicy"' 'genpolicyPath = "${genpolicy}/bin/genpolicy"'
    '';
  };

  push-coordinator = pushContainer coordinator;
  push-initializer = pushContainer initializer;

  azure-cli-with-extensions = callPackage ./azurecli.nix { };

  create-coco-aks = writeShellApplication {
    name = "create-coco-aks";
    runtimeInputs = [ azure-cli-with-extensions ];
    text = builtins.readFile ./create-coco-aks.sh;
  };
  destroy-coco-aks = writeShellApplication {
    name = "destroy-coco-aks";
    runtimeInputs = [ azure-cli-with-extensions ];
    text = ''az group delete --name "$1"'';
  };

  generate = pkgs.writeShellApplication {
    name = "generate";
    runtimeInputs = with pkgs; [ go protobuf protoc-gen-go protoc-gen-go-grpc ];
    text = ''
      go generate ./...
      go mod tidy
    '';
  };

  genpolicy = genpolicy-msft;
  genpolicy-msft = callPackage ./genpolicy_msft.nix { };
  genpolicy-kata = callPackage ./genpolicy_kata.nix { };
}
