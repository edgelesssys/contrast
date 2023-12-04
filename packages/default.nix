{ pkgs
, goVendorHash
}:

with pkgs;

let
  goFiles = lib.fileset.unions [
    ../go.mod
    ../go.sum
    (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) ../.)
  ];

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
      flags = [ "-trimpath" ];
      proxyVendor = true;
      vendorHash = goVendorHash;
      meta.mainProgram = "${subpackage}";
    } // attrs))
    { };

  buildContainer = drv: pkgs.dockerTools.buildImage {
    name = drv.name;
    tag = "latest";
    copyToRoot = with pkgs.dockerTools; [
      caCertificates
    ];
    config = {
      Cmd = [ "${lib.getExe drv}" ];
    };
  };

  pushContainer = name: container: pkgs.writeShellApplication {
    name = "push";
    runtimeInputs = with pkgs; [ crane gzip ];
    text = ''
      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT
      gunzip < "${container}" > "$tmpdir/image.tar"
      crane push "$tmpdir/image.tar" "${name}"
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

  push-coordinator = pushContainer "ghcr.io/katexochen/coordinator-kbs:latest" coordinator;
  push-initializer = pushContainer "ghcr.io/katexochen/initializer:latest" initializer;

  generate = pkgs.writeShellApplication {
    name = "generate";
    runtimeInputs = with pkgs; [ go protobuf protoc-gen-go protoc-gen-go-grpc ];
    text = ''go generate ./...'';
  };

  genpolicy = genpolicy-msft;
  genpolicy-msft = callPackage ./genpolicy_msft.nix { };
  genpolicy-kata = callPackage ./genpolicy_kata.nix { };
}
