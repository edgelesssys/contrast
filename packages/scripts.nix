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
      contrast
      microsoft.genpolicy
    ];
    text = ''
      imageRef=$1

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      echo "ghcr.io/edgelesssys/contrast/coordinator:latest=$imageRef" > "$tmpdir/image-replacements.txt"
      resourcegen --image-replacements "$tmpdir/image-replacements.txt" --add-load-balancers coordinator > "$tmpdir/coordinator_base.yml"

      pushd "$tmpdir" >/dev/null
      cp ${pkgs.microsoft.genpolicy.rules-coordinator}/genpolicy-rules.rego rules.rego
      cp ${pkgs.microsoft.genpolicy.settings-coordinator}/genpolicy-settings.json .
      genpolicy < "$tmpdir/coordinator_base.yml"
      popd >/dev/null
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

  get-azure-node-image-version = writeShellApplication {
    name = "get-azure-node-image-version";
    runtimeInputs = with pkgs; [
      azure-cli
      jq
    ];
    text = ''
      set -euo pipefail

      name=""
      pool="nodepool2"

      for i in "$@"; do
        case $i in
        --name=*) name="''${i#*=}"; shift ;;
        --pool=*) pool="''${i#*=}"; shift ;;
        *) echo "Unknown option $i"; exit 1 ;;
        esac
      done

      az aks nodepool show \
        --resource-group "$name" \
        --cluster-name "$name" \
        --name "$pool" \
        | jq -r '.nodeImageVersion'
    '';
  };

  update-contrast-releases = writeShellApplication {
    name = "update-contrast-releases";
    runtimeInputs = with pkgs; [
      jq
    ];
    text = builtins.readFile ./update-contrast-releases.sh;
  };

  update-release-urls = writeShellApplication {
    name = "update-release-urls";
    runtimeInputs = with pkgs; [ coreutils findutils gnused ];
    text = ''
      tag="[a-zA-Z0-9_.-]\{1,\}"
      sha="@sha256:[a-fA-F0-9]\{64\}"

      while IFS= read -r line; do
        image_source=$(echo "$line" | sed "s/:.*//" | sed "s/\./\\\./g")
        image_target=$(echo "$line" | cut -d"=" -f2)
        expr="$image_source\(:$tag\($sha\)\?\|$sha\)"
        find "./docs/versioned_docs/version-$MAJOR_MINOR" -type f -exec sed -i "s#$expr#$image_target#g" {} \;
      done <"../image-replacements.txt"

      link_source="github\.com/edgelesssys/contrast/releases/\(latest/download\|download/$tag\)/"
      link_target="github\.com/edgelesssys/contrast/releases/download/$VERSION/"
      find "./docs/versioned_docs/version-$MAJOR_MINOR" -type f -exec sed -i "s#$link_source#$link_target#g" {} \;
    '';
  };

  # Temporary workaround until the sync server is stateful.
  renew-sync-fifo = writeShellApplication {
    name = "renew-sync-fifo";
    runtimeInputs = with pkgs; [ kubectl ];
    text = ''
      kubectl delete configmap sync-server-fifo || true
      syncIP=$(kubectl get svc sync -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
      fifoUUID=$(curl -fsSL "$syncIP:8080/fifo/new" | jq -r '.uuid')
      kubectl create configmap sync-server-fifo --from-literal=uuid="$fifoUUID"
    '';
  };

  # Usage: cat events.log | parse-blocked-by-policy
  parse-blocked-by-policy = writeShellApplication {
    name = "parse-blocked-by-policy";
    runtimeInputs = with pkgs; [ gnugrep gnused ];
    text = ''
      set -euo pipefail
      grep "CreateContainerRequest is blocked by policy" |
      sed 's/ agent_policy:/\nagent_policy:/g' |
      sed 's/\\"/"/g'
    '';
  };

  # Usage: cat deployment.yml | extract-policies
  extract-policies = writeShellApplication {
    name = "extract-policies";
    runtimeInputs = with pkgs; [ yq-go ];
    text = ''
      set -euo pipefail
      while read -r line; do
          name=$(echo "$line" | cut -d' ' -f1)
          namespace=$(echo "$line" | cut -d' ' -f2)
          echo "Extracting policy for $namespace.$name" >&2
          echo "$line" | cut -d' ' -f3 | base64 -d > "$namespace.$name.rego"
      done < <(
        yq '.metadata.name
          + " "
          + .metadata.namespace
          // "default"
          + " "
          + .spec.template.metadata.annotations["io.katacontainers.config.agent.policy"]
          // .metadata.annotations["io.katacontainers.config.agent.policy"]'
      )
    '';
  };
}
