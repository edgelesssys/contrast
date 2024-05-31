# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ pkgs
, writeShellApplication
}:

{
  create-coco-aks = writeShellApplication {
    name = "create-coco-aks";
    runtimeInputs = with pkgs; [ azure-cli ];
    text = builtins.readFile ./create-coco-aks.sh;
  };

  destroy-coco-aks = writeShellApplication {
    name = "destroy-coco-aks";
    runtimeInputs = with pkgs; [ azure-cli ];
    text = builtins.readFile ./destroy-coco-aks.sh;
  };

  generate = writeShellApplication {
    name = "generate";
    runtimeInputs = with pkgs; [
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
    runtimeInputs = with pkgs; [ go govulncheck ];
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
    runtimeInputs = with pkgs; [ go golangci-lint ];
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
    runtimeInputs = with pkgs; [
      crane
      kypatch
      jq
    ];
    text = ''
      targetPath=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      gunzip < "${pkgs.containers.coordinator}" > "$tmpdir/coordinator.tar"
      gunzip < "${pkgs.containers.initializer}" > "$tmpdir/initializer.tar"
      gunzip < "${pkgs.containers.openssl}" > "$tmpdir/openssl.tar"
      gunzip < "${pkgs.containers.port-forwarder}" > "$tmpdir/port-forwarder.tar"
      gunzip < "${pkgs.containers.service-mesh-proxy}" > "$tmpdir/service-mesh-proxy.tar"

      coordHash=$(crane digest --tarball "$tmpdir/coordinator.tar")
      initHash=$(crane digest --tarball "$tmpdir/initializer.tar")
      opensslHash=$(crane digest --tarball "$tmpdir/openssl.tar")
      forwarderHash=$(crane digest --tarball "$tmpdir/port-forwarder.tar")
      serviceMeshProxyHash=$(crane digest --tarball "$tmpdir/service-mesh-proxy.tar")
      nodeInstallerHash=$(jq -r '.manifests[0].digest' "${pkgs.contrast-node-installer-image}/index.json")

      kypatch images "$targetPath" \
        --replace "contrast/coordinator:latest" "contrast/coordinator@$coordHash" \
        --replace "contrast/initializer:latest" "contrast/initializer@$initHash" \
        --replace "contrast/openssl:latest" "contrast/openssl@$opensslHash" \
        --replace "contrast/port-forwarder:latest" "contrast/port-forwarder@$forwarderHash" \
        --replace "contrast/service-mesh-proxy:latest" "contrast/service-mesh-proxy@$serviceMeshProxyHash" \
        --replace "contrast/node-installer:latest" "contrast/node-installer@$nodeInstallerHash"
    '';
  };

  kubectl-wait-ready = writeShellApplication {
    name = "kubectl-wait-ready";
    runtimeInputs = with pkgs; [ kubectl ];
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
    runtimeInputs = with pkgs; [ iproute2 ];
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
    runtimeInputs = with pkgs; [
      yq-go
      genpolicy-msft
      contrast
    ];
    text = ''
      imageRef=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      echo "ghcr.io/edgelesssys/contrast/coordinator:latest=$imageRef" > "$tmpdir/image-replacements.txt"
      resourcegen --image-replacements "$tmpdir/image-replacements.txt" --add-load-balancers coordinator > "$tmpdir/coordinator_base.yml"

      pushd "$tmpdir" >/dev/null
      cp ${pkgs.genpolicy-msft.rules-coordinator}/genpolicy-rules.rego rules.rego
      cp ${pkgs.genpolicy-msft.settings}/genpolicy-settings.json .
      genpolicy < "$tmpdir/coordinator_base.yml"
      popd >/dev/null
    '';
  };

  fetch-latest-contrast = writeShellApplication {
    name = "fetch-latest-contrast";
    runtimeInputs = with pkgs; [
      jq
      github-cli
      yq-go
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
    runtimeInputs = with pkgs; [
      azure-cli
      jq
    ];
    text = ''
      sku=''${1:-Standard_DC4as_cc_v5}
      az vm list-skus --size "$sku" | jq -r '.[] | .locations.[]'
    '';
  };

  update-versions-json = writeShellApplication {
    name = "update-versions-json";
    runtimeInputs = with pkgs; [
      jq
    ];
    text = builtins.readFile ./update-versions-json.sh;
  };
}
