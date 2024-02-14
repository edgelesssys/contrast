{ lib, pkgs }:

with pkgs;

let
  pkgs' = pkgs // by-name;

  by-name = lib.by-name pkgs' ./by-name;

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

  version = builtins.readFile ../version.txt;
in

with by-name;

rec {
  cli-release = (nunki.override (prevArgs: {
    ldflags = prevArgs.ldflags ++ [
      "-X main.DefaultCoordinatorPolicyHash=${builtins.readFile ../cli/assets/coordinator-policy-hash}"
    ];
  })).cli;

  coordinator = dockerTools.buildImage {
    name = "coordinator";
    tag = "v${version}";
    copyToRoot = with dockerTools; [ caCertificates ];
    config = {
      Cmd = [ "${nunki.coordinator}/bin/coordinator" ];
      Env = [ "PATH=/bin" ]; # This is only here for policy generation.
    };
  };
  initializer = dockerTools.buildImage {
    name = "initializer";
    tag = "v${version}";
    copyToRoot = with dockerTools; [ caCertificates ];
    config = {
      Cmd = [ "${nunki.initializer}/bin/initializer" ];
      Env = [ "PATH=/bin" ]; # This is only here for policy generation.
    };
  };

  opensslContainer = dockerTools.buildImage {
    name = "openssl";
    tag = "v${version}";
    copyToRoot = [ openssl bash coreutils ncurses bashInteractive vim procps ];
    config = {
      Cmd = [ "bash" ];
      Env = [ "PATH=/bin" ]; # This is only here for policy generation.
    };
  };
  port-forwarder = dockerTools.buildImage {
    name = "port-forwarder";
    tag = "v${version}";
    copyToRoot = [ bash socat ];
  };

  push-coordinator = pushContainer coordinator;
  push-initializer = pushContainer initializer;

  push-openssl = pushContainer opensslContainer;
  push-port-forwarder = pushContainer port-forwarder;

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
      nix-update --version=skip --flake nunki.cli
    '';
  };

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
      opensslContainer
      port-forwarder
    ];
    text = ''
      targetPath=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      gunzip < "${coordinator}" > "$tmpdir/coordinator.tar"
      gunzip < "${initializer}" > "$tmpdir/initializer.tar"
      gunzip < "${opensslContainer}" > "$tmpdir/openssl.tar"
      gunzip < "${port-forwarder}" > "$tmpdir/port-forwarder.tar"

      coordHash=$(crane digest --tarball "$tmpdir/coordinator.tar")
      initHash=$(crane digest --tarball "$tmpdir/initializer.tar")
      opensslHash=$(crane digest --tarball "$tmpdir/openssl.tar")
      forwarderHash=$(crane digest --tarball "$tmpdir/port-forwarder.tar")

      kypatch images "$targetPath" \
        --replace "nunki/coordinator:latest" "nunki/coordinator@$coordHash" \
        --replace "nunki/initializer:latest" "nunki/initializer@$initHash" \
        --replace "nunki/openssl:latest" "nunki/openssl@$opensslHash" \
        --replace "nunki/port-forwarder:latest" "nunki/port-forwarder@$forwarderHash"
    '';
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

  # write-coordinator-yaml prints a Nunki Coordinator deployment including the default policy.
  # It's intended for two purposes: (1) releasing a portable coordinator.yaml and (2) updating the embedded policy hash.
  write-coordinator-yaml = writeShellApplication {
    name = "write-coordinator-yaml";
    runtimeInputs = [
      yq-go
      genpolicy
    ];
    text = ''
      imageRef=$1:v${version}

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      # TODO(burgerdev): consider a dedicated coordinator template instead of the simple one
      yq < deployments/simple/coordinator.yml > "$tmpdir/coordinator.yml" \
        "del(.metadata.namespace) | (select(.kind == \"Deployment\") | .spec.template.spec.containers[0].image) = \"$imageRef\""

      pushd "$tmpdir" >/dev/null
      cp ${genpolicy.settings}/genpolicy-settings.json .
      cp ${genpolicy.rules-coordinator}/genpolicy-rules.rego rules.rego
      genpolicy < "$tmpdir/coordinator.yml"
      popd >/dev/null
    '';
  };
} // by-name
