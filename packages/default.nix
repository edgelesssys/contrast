{ pkgs
, version
}:

with pkgs;

let
  # The source of our local Go module. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  goFiles = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum
      (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) ../.)
    ];
  };

  pushContainer = container: writeShellApplication {
    name = "push";
    runtimeInputs = [ crane gzip ];
    text = ''
      imageName="$1"
      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT
      gunzip < "${container}" > "$tmpdir/image.tar"
      crane push "$tmpdir/image.tar" "$imageName:${container.imageTag}"
    '';
  };
in

rec {
  nunki =
    let
      subPackages = [ "coordinator" "initializer" "cli" ];
    in
    buildGoModule {
      inherit version subPackages;
      name = "nunki";

      outputs = subPackages ++ [ "out" ];

      src = goFiles;
      proxyVendor = true;
      vendorHash = "sha256-dkFAlqAzVD82yWrrdscZumY4hP/XP3hn8CuZ0tkZuhg=";

      prePatch = ''
        install -D ${genpolicy.settings-dev}/genpolicy-settings.json cli/assets/genpolicy-settings.json
        install -D ${genpolicy.rules}/genpolicy-rules.rego cli/assets/genpolicy-rules.rego
      '';

      CGO_ENABLED = 0;
      ldflags = [
        "-s"
        "-w"
        "-buildid="
        "-X main.genpolicyPath=${genpolicy}/bin/genpolicy"
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
          install -Dm755 "$out/bin/$sub" "''${!sub}/bin/$sub"
        done
      '';
      meta.mainProgram = "cli";
    };
  inherit (nunki) cli;

  coordinator = dockerTools.buildImage {
    name = "coordinator";
    tag = "v${version}";
    copyToRoot = with dockerTools; [ caCertificates ];
    config = {
      Cmd = [ "${nunki.coordinator}/bin/coordinator" ];
    };
  };
  initializer = dockerTools.buildImage {
    name = "initializer";
    tag = "v${version}";
    copyToRoot = with dockerTools; [ caCertificates ];
    config = {
      Cmd = [ "${nunki.initializer}/bin/initializer" ];
    };
  };

  push-coordinator = pushContainer coordinator;
  push-initializer = pushContainer initializer;

  push-openssl = pushContainer (dockerTools.buildImage {
    name = "openssl";
    tag = "v${version}";
    copyToRoot = [ openssl bash coreutils ncurses bashInteractive vim procps ];
    config = {
      Cmd = [ "bash" ];
    };
  });

  azure-cli-with-extensions = callPackage ./azurecli.nix { };

  create-coco-aks = writeShellApplication {
    name = "create-coco-aks";
    runtimeInputs = [ azure-cli-with-extensions ];
    text = builtins.readFile ./create-coco-aks.sh;
  };
  destroy-coco-aks = writeShellApplication {
    name = "destroy-coco-aks";
    runtimeInputs = [ azure-cli-with-extensions ];
    text = builtins.readFile ./destroy-coco-aks.sh;
  };

  generate = writeShellApplication {
    name = "generate";
    runtimeInputs = [
      go
      protobuf
      protoc-gen-go
      protoc-gen-go-grpc
      nix-update
    ];
    text = ''
      go mod tidy
      go generate ./...

      # All binaries of the local Go module share the same builder,
      # we only need to update one of them to update the vendorHash
      # of the builder.
      nix-update --version=skip --flake cli
    '';
  };

  genpolicy = genpolicy-msft;
  genpolicy-msft = pkgsStatic.callPackage ./genpolicy_msft.nix { };
  genpolicy-kata = pkgsStatic.callPackage ./genpolicy_kata.nix { };

  govulncheck = writeShellApplication {
    name = "govulncheck";
    runtimeInputs = [ go pkgs.govulncheck ];
    text = ''govulncheck "$@"'';
  };

  golangci-lint = writeShellApplication {
    name = "golangci-lint";
    runtimeInputs = [ go pkgs.golangci-lint ];
    text = ''golangci-lint "$@"'';
  };

  patch-nunki-image-hashes = writeShellApplication {
    name = "patch-nunki-image-hashes";
    runtimeInputs = [
      coordinator
      crane
      initializer
      kypatch
    ];
    text = ''
      targetPath=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      gunzip < "${coordinator}" > "$tmpdir/coordinator.tar"
      gunzip < "${initializer}" > "$tmpdir/initializer.tar"

      coordHash=$(crane digest --tarball "$tmpdir/coordinator.tar")
      initHash=$(crane digest --tarball "$tmpdir/initializer.tar")

      kypatch images "$targetPath" \
        --replace "nunki/coordinator:latest" "nunki/coordinator@$coordHash" \
        --replace "nunki/initializer:latest" "nunki/initializer@$initHash"
    '';
  };

  kypatch = writeShellApplication {
    name = "kypatch";
    runtimeInputs = [ yq-go ];
    text = builtins.readFile ./kypatch.sh;
  };

  kubectl-wait-ready = writeShellApplication {
    name = "kubectl-wait-ready";
    runtimeInputs = [ kubectl ];
    text = ''
      namespace=$1
      name=$2

      echo "Waiting for $name.$namespace to become ready" >&2

      timeout=180

      interval=4
      while [ $timeout -gt 0 ]; do
        if kubectl -n "$namespace" get pods -o custom-columns=LABELS:.metadata.labels | grep -q "app.kubernetes.io/name:$name"; then
          break
        fi
        sleep "$interval"
        timeout=$((timeout - interval))
      done

      kubectl wait \
         --namespace "$namespace" \
         --selector "app.kubernetes.io/name=$name" \
         --for=condition=Ready \
         --timeout="''${timeout}s" \
         pods
    '';
  };

  wait-for-port-listen = writeShellApplication {
    name = "wait-for-port-listen";
    runtimeInputs = [ iproute2 ];
    text = ''
      port=$1

      function ss-listen-on-port() {
        ss \
          --tcp \
          --numeric \
          --listening \
          --no-header \
          --ipv4 \
          src ":$port"
      }

      tries=15 # 3 seconds
      interval=0.2

      while [[ "$tries" -gt 0 ]]; do
        if [[ -n $(ss-listen-on-port) ]]; then
          exit 0
        fi
        sleep "$interval"
        tries=$((tries - 1))
      done

      echo "Port $port did not reach state LISTENING" >&2
      exit 1
    '';
  };
}
