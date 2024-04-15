{ pkgs }:

with pkgs;

{
  create-coco-aks = writeShellApplication {
    name = "create-coco-aks";
    runtimeInputs = [ azure-cli ];
    text = builtins.readFile ./create-coco-aks.sh;
  };

  destroy-coco-aks = writeShellApplication {
    name = "destroy-coco-aks";
    runtimeInputs = [ azure-cli ];
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
      while IFS= read -r dir; do
        echo "Running go mod tidy on $dir" >&2
        go mod tidy
        echo "Running go generate on $dir" >&2
        go generate ./...
      done < <(go list -f '{{.Dir}}' -m)

      # All binaries of the main Go module share the same builder,
      # we only need to update one of them to update the vendorHash
      # of the builder.
      echo "Updating vendorHash of contrast.cli package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.contrast.cli

      echo "Updating vendorHash of service-mesh package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.service-mesh

      echo "Updating vendorHash of node-installer package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.contrast-node-installer

      echo "Updateing yarn offlineCache hash of contrast-docs package" >&2
      nix-update --version=skip --flake \
        --override-filename=packages/by-name/contrast-docs/package.nix \
        legacyPackages.x86_64-linux.contrast-docs
    '';
  };

  govulncheck = writeShellApplication {
    name = "govulncheck";
    runtimeInputs = [ go pkgs.govulncheck ];
    text = ''
      exitcode=0

      while IFS= read -r dir; do
        echo "Running govulncheck on $dir"
        govulncheck -C "$dir" || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      exit $exitcode
    '';
  };

  golangci-lint = writeShellApplication {
    name = "golangci-lint";
    runtimeInputs = [ go pkgs.golangci-lint ];
    text = ''
      exitcode=0

      while IFS= read -r dir; do
        echo "Running golangci-lint on $dir" >&2
        golangci-lint run "$dir/..." || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      echo "Verifying golangci-lint config" >&2
      golangci-lint config verify || exitcode=$?

      exit $exitcode
    '';
  };

  patch-contrast-image-hashes = writeShellApplication {
    name = "patch-contrast-image-hashes";
    runtimeInputs = [
      crane
      kypatch
    ];
    text = ''
      targetPath=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      gunzip < "${containers.coordinator}" > "$tmpdir/coordinator.tar"
      gunzip < "${containers.initializer}" > "$tmpdir/initializer.tar"
      gunzip < "${containers.openssl}" > "$tmpdir/openssl.tar"
      gunzip < "${containers.port-forwarder}" > "$tmpdir/port-forwarder.tar"
      gunzip < "${containers.service-mesh-proxy}" > "$tmpdir/service-mesh-proxy.tar"

      coordHash=$(crane digest --tarball "$tmpdir/coordinator.tar")
      initHash=$(crane digest --tarball "$tmpdir/initializer.tar")
      opensslHash=$(crane digest --tarball "$tmpdir/openssl.tar")
      forwarderHash=$(crane digest --tarball "$tmpdir/port-forwarder.tar")
      serviceMeshProxyHash=$(crane digest --tarball "$tmpdir/service-mesh-proxy.tar")

      kypatch images "$targetPath" \
        --replace "contrast/coordinator:latest" "contrast/coordinator@$coordHash" \
        --replace "contrast/initializer:latest" "contrast/initializer@$initHash" \
        --replace "contrast/openssl:latest" "contrast/openssl@$opensslHash" \
        --replace "contrast/port-forwarder:latest" "contrast/port-forwarder@$forwarderHash" \
        --replace "contrast/service-mesh-proxy:latest" "contrast/service-mesh-proxy@$serviceMeshProxyHash"
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

  # write-coordinator-yaml prints a Contrast Coordinator deployment including the default policy.
  # It's intended for two purposes: (1) releasing a portable coordinator.yml and (2) updating the embedded policy hash.
  write-coordinator-yaml = writeShellApplication {
    name = "write-coordinator-policy";
    runtimeInputs = [
      yq-go
      genpolicy-msft
      contrast
    ];
    text = ''
      imageRef=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      resourcegen coordinator-release "$tmpdir/coordinator_base.yml"
      yq < "$tmpdir/coordinator_base.yml" > "$tmpdir/coordinator.yml" \
         "(select(.kind == \"Deployment\") | .spec.template.spec.containers[0].image) = \"$imageRef\""

      pushd "$tmpdir" >/dev/null
      cp ${genpolicy-msft.rules-coordinator}/genpolicy-rules.rego rules.rego
      cp ${genpolicy-msft.settings}/genpolicy-settings.json .
      genpolicy < "$tmpdir/coordinator.yml"
      popd >/dev/null
    '';
  };

  fetch-latest-contrast = writeShellApplication {
    name = "fetch-latest-contrast";
    runtimeInputs = [
      jq
      github-cli
    ];
    text = ''
      namespace=$1
      targetDir=$2
      release=$(gh release list --json name,isLatest | jq -r '.[] | select(.isLatest) | .name')
      gh release download "$release" \
        --repo edgelesssys/contrast \
        -D "$targetDir" \
        --skip-existing
      chmod a+x "$targetDir/contrast"

      yq -i ".metadata.namespace = \"$namespace\"" "$targetDir/coordinator.yml"
    '';
  };

  get-azure-sku-locations = writeShellApplication {
    name = "get-azure-sku-locations";
    runtimeInputs = [
      azure-cli
      jq
    ];
    text = ''
      sku=''${1:-Standard_DC4as_cc_v5}
      az vm list-skus --size "$sku" | jq -r '.[] | .locations.[]'
    '';
  };
}
