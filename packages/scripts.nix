# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ pkgs, writeShellApplication }:

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

  upload-image = writeShellApplication {
    name = "upload-image";
    runtimeInputs = with pkgs; [
      azure-cli
      gnused
      uplosi
    ];
    text =
      let
        image = pkgs.image-podvm;
      in
      ''
        subscriptionId=""
        location="GermanyWestCentral"
        resourceGroup=""

        for i in "$@"; do
          case $i in
          --subscription-id=*)
            subscriptionId="''${i#*=}"
            shift
            ;;
          --location=*)
            location="''${i#*=}"
            shift
            ;;
          --resource-group=*)
            resourceGroup="''${i#*=}"
            shift
            ;;
          *)
            echo "Unknown option $i"
            exit 1
            ;;
          esac
        done

        set -x

        # Create a unique, semver compatible version.
        imageVersion=0.0.$(date "+%s")

        # Create an uplosi config.
        cat <<EOF > uplosi.conf
        [base]
        imageVersion = "''${imageVersion}"
        name = "contrast"
        provider = "azure"

        [base.azure]
        subscriptionID = "''${subscriptionId}"
        location = "''${location}"
        resourceGroup = "''${resourceGroup}"
        sharedImageGallery = "''${resourceGroup//-/_}_contrast"
        sharingProfile = "private"
        EOF

        imageCacheDir="''${CONTRAST_CACHE_DIR}"/image-upload
        mkdir -p "''${imageCacheDir}"

        cacheFile="''${imageCacheDir}"/${builtins.baseNameOf image}.image-id
        # Check if th image has been cached.
        if [[ ! -f "$cacheFile" ]]; then
          # Upload the image.
          image_id=$(uplosi upload ${image}/*.raw)
          echo "$image_id" > "$cacheFile"
        else
          # Use the image id in the cache.
          image_id=$(cat "$cacheFile")
        fi

        # Store the image id in a terraform variable.
        echo "image_id = \"''${image_id}\"" > infra/azure-peerpods/image_id.auto.tfvars
      '';
  };

  generate = writeShellApplication {
    name = "generate";
    runtimeInputs = with pkgs; [
      go
      protobuf
      protoc-gen-go
      protoc-gen-go-grpc
      nix-update
      scripts.go-directive-sync
    ];
    text = ''
      echo "Syncing go directive versions in go.mod/go.work files" >&2
      go-directive-sync

      while IFS= read -r dir; do
        echo "Running go mod tidy on $dir" >&2
        go mod -C "$dir" tidy
        echo "Running go generate on $dir" >&2
        go generate -C "$dir" ./...
      done < <(go list -f '{{.Dir}}' -m)

      protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./internal/userapi/userapi.proto

      # Notice: Order matters! Packages must be updated before their dependents.
      echo "Updating vendorHash of tdx-measure package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.tdx-measure
      echo "Updating vendorHash of service-mesh package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.service-mesh
      echo "Updating vendorHash of snp-id-block-generator package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.snp-id-block-generator
      echo "Updating vendorHash of contrast.cli package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.contrast
      echo "Updating vendorHash of nydus-pull package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.nydus-pull

      echo "Updating src hash of kata.kata-kernel-uvm.configfile" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.kata.kata-kernel-uvm.configfile

      echo "Updateing yarn offlineCache hash of contrast-docs package" >&2
      nix-update --version=skip --flake \
        --override-filename=packages/by-name/contrast-docs/package.nix \
        legacyPackages.x86_64-linux.contrast-docs
    '';
  };

  govulncheck = writeShellApplication {
    name = "govulncheck";
    runtimeInputs = with pkgs; [
      go
      govulncheck
    ];
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
    runtimeInputs = with pkgs; [
      go
      golangci-lint
    ];
    text = ''
      exitcode=0

      while IFS= read -r dir; do
        echo "Running golangci-lint on $dir" >&2
        golangci-lint run "$dir/..." || exitcode=$?
        golangci-lint run --build-tags enterprise "$dir/..." || exitcode=$?
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
    ];
    text = ''
      imageRef=$1
      platform=$2

      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT

      echo "ghcr.io/edgelesssys/contrast/coordinator:latest=$imageRef" > "$tmpdir/image-replacements.txt"
      resourcegen --platform "$platform" --image-replacements "$tmpdir/image-replacements.txt" --add-load-balancers coordinator > "$tmpdir/coordinator_base.yml"

      pushd "$tmpdir" >/dev/null

      case "$platform" in
        "aks-clh-snp")
          cp ${pkgs.microsoft.genpolicy.rules-coordinator}/genpolicy-rules.rego rules.rego
          cp ${pkgs.microsoft.genpolicy.settings-coordinator}/genpolicy-settings.json .
          ${pkgs.microsoft.genpolicy}/bin/genpolicy < "$tmpdir/coordinator_base.yml"
        ;;
        "metal-qemu-snp"|"metal-qemu-snp-gpu"|"k3s-qemu-snp"|"k3s-qemu-snp-gpu"|"metal-qemu-tdx"|"k3s-qemu-tdx"|"rke2-qemu-tdx")
          cp ${pkgs.kata.genpolicy.rules-coordinator}/genpolicy-rules.rego rules.rego
          cp ${pkgs.kata.genpolicy.settings-coordinator}/genpolicy-settings.json .
          ${pkgs.kata.genpolicy}/bin/genpolicy < "$tmpdir/coordinator_base.yml"
        ;;
        *)
          echo "Unsupported platform: $platform" >&2
          exit 1
        ;;
      esac

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
      pool="nodepool1"

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
    runtimeInputs = with pkgs; [ jq ];
    text = builtins.readFile ./update-contrast-releases.sh;
  };

  update-release-urls = writeShellApplication {
    name = "update-release-urls";
    runtimeInputs = with pkgs; [
      coreutils
      findutils
      gnused
    ];
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
    runtimeInputs = with pkgs; [
      gnugrep
      gnused
    ];
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
          if [[ "$line" == "---" ]]; then
              continue
          fi
          name=$(echo "$line" | cut -d' ' -f1)
          namespace=$(echo "$line" | cut -d' ' -f2)
          echo "Extracting policy for $namespace.$name" >&2
          echo "$line" | cut -d' ' -f3 | base64 -d > "$namespace.$name.rego"
      done < <(
        yq '.metadata.name
          + " " + (
            .metadata.namespace
            // "default"
          ) + " " + (
            .spec.template.metadata.annotations["io.katacontainers.config.agent.policy"]
            // .metadata.annotations["io.katacontainers.config.agent.policy"]
          )'
      )
    '';
  };

  merge-kube-config = writeShellApplication {
    name = "merge-kube-config";
    runtimeInputs = with pkgs; [ yq-go ];
    text = ''
      set -euo pipefail
      mergedConfig=$(mktemp)
      KUBECONFIG_BAK=''${KUBECONFIG:-~/.kube/config}
      KUBECONFIG=$1:''${KUBECONFIG_BAK} kubectl config view --flatten > "$mergedConfig"
      newContext=$(yq -r '.contexts.[0].name' "$1")
      declare -x newContext
      yq -i '.current-context = env(newContext)' "$mergedConfig"
      targetFile="''${KUBECONFIG_BAK%%:*}"
      mkdir -p "$(dirname "$targetFile")"
      mv "$mergedConfig" "$targetFile"
    '';
  };

  # Usage: get-credentials $gcloudSecretRef
  get-credentials = writeShellApplication {
    name = "extract-policies";
    runtimeInputs = with pkgs; [
      google-cloud-sdk
      scripts.merge-kube-config
    ];
    text = ''
      set -euo pipefail
      tmpConfig=$(mktemp)
      gcloud secrets versions access "$1" --out-file="$tmpConfig"
      merge-kube-config "$tmpConfig"
    '';
  };

  # Usage: get-logs [start | download] $namespaceFile
  get-logs = writeShellApplication {
    name = "get-logs";
    runtimeInputs = with pkgs; [
      kubectl
    ];
    text = ''
      set -euo pipefail

      if [[ $# -lt 2 ]]; then
        echo "Usage: get-logs [start | download] namespaceFile"
        exit 1
      fi
      case $1 in
      start)
        while ! [[ -s "$2" ]]; do
          sleep 1
        done
        namespace="$(head -n1 "$2")"
        cp ./packages/log-collector.yaml ./workspace/log-collector.yaml
        sed -i "s/@@NAMESPACE@@/''${namespace}/g" ./workspace/log-collector.yaml
        kubectl apply -f ./workspace/log-collector.yaml 1>/dev/null 2>/dev/null
        ;;
      download)
        namespace="$(head -n1 "$2")"
        pod="$(kubectl get pods -o name -n "$namespace" | grep log-collector | cut -c 5-)"
        mkdir -p ./workspace/logs
        kubectl wait --for=condition=Ready -n "$namespace" "pod/$pod"
        kubectl exec -n "$namespace" "$pod" -- /bin/bash -c "rm -f /exported-logs.tar.gz; cp -r /export /export-no-stream; tar zcvf /exported-logs.tar.gz /export-no-stream; rm -rf /export-no-stream"
        kubectl cp -n "$namespace" "$pod:/exported-logs.tar.gz" ./workspace/logs/exported-logs.tar.gz
        tar xzvf ./workspace/logs/exported-logs.tar.gz --directory ./workspace/logs
        ;;
      *)
        echo "Unknown option $1"
        echo "Usage: get-logs [start | download] namespaceFile"
        exit 1
      esac
    '';
  };

  deploy-caa = writeShellApplication {
    name = "deploy-caa";
    runtimeInputs = with pkgs; [ kubectl ];
    text = ''
      set -euo pipefail

      for i in "$@"; do
        case $i in
        --kustomization=*)
          kustomizationFile="''${i#*=}"
          shift
          ;;
        --pub-key=*)
          pubKeyFile="''${i#*=}"
          shift
          ;;
        *)
          echo "Unknown option $i"
          exit 1
          ;;
        esac
      done

      tmpdir=$(mktemp -d)
      cp -r ${pkgs.cloud-api-adaptor.src}/src/cloud-api-adaptor/install/* "$tmpdir"
      chmod -R +w "$tmpdir"
      cp "$kustomizationFile" "$tmpdir/overlays/azure/kustomization.yaml"
      cp "$pubKeyFile" "$tmpdir/overlays/azure/id_rsa.pub"

      kubectl apply -k "github.com/confidential-containers/operator/config/release?ref=v${pkgs.cloud-api-adaptor.version}"
      kubectl apply -k "github.com/confidential-containers/operator/config/samples/ccruntime/peer-pods?ref=v${pkgs.cloud-api-adaptor.version}"
      kubectl apply -k "$tmpdir/overlays/azure"
    '';
  };

  cleanup-bare-metal = writeShellApplication {
    name = "cleanup-bare-metal";
    runtimeInputs = with pkgs; [
      busybox
      kubectl
      dasel
      scripts.cleanup-images
    ];
    text = builtins.readFile ./cleanup-bare-metal.sh;
  };

  cleanup-images = writeShellApplication {
    name = "cleanup-images";
    runtimeInputs = with pkgs; [
      gnugrep
      busybox
      containerd
    ];
    text = builtins.readFile ./cleanup-images.sh;
  };

  test-peerpods = writeShellApplication {
    name = "test-peerpods";
    runtimeInputs = with pkgs; [
      just
      azure-cli
      kubectl
    ];
    text = builtins.readFile ./test-peerpods.sh;
  };

  # Sync the go directive between go.mod/go.work files (that is, the 'go' statement of these files).
  # We take the latest version we find and use that everywhere.
  go-directive-sync = writeShellApplication {
    name = "go-directive-sync";
    runtimeInputs = with pkgs; [
      go
      findutils
      coreutils
    ];
    text = ''
      set -euo pipefail

      modFiles=$(find . -regex '.*/go[.]\(mod\|work\)$')

      goVers=()
      while IFS= read -r f; do
        ver=$(grep -E '^go [0-9]+[.][0-9]+[.][0-9]+$' "$f")
        goVers+=("$ver")
      done <<< "$modFiles"

      maxVer=$(printf "%s\n" "''${goVers[@]}" | sort -V | tail -n1)

      while IFS= read -r f; do
        sed -i "s/^go [0-9]\+\.[0-9]\+\.[0-9]\+$/''${maxVer}/" "$f"
      done <<< "$modFiles"
    '';
  };
}
