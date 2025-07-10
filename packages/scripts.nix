# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  writeShellApplication,
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
      scripts.go-directive-sync
    ];
    text = ''
      echo "Syncing go directive versions in go.mod/go.work files" >&2
      go-directive-sync

      while IFS= read -r dir; do
        echo "Running go mod tidy on $dir" >&2
        go mod -C "$dir" tidy

        # go mod tidy bumps the go version if a dependency requires a newer one.
        # We need to run go-directive-sync again to sync the go version.
        go-directive-sync

        echo "Running go generate on $dir" >&2
        go generate -C "$dir" ./...
      done < <(go list -f '{{.Dir}}' -m)

      # Notice: Order matters! Packages must be updated before their dependents.
      echo "Updating vendorHash of tdx-measure package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.tdx-measure
      echo "Updating vendorHash of service-mesh package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.service-mesh
      echo "Updating vendorHash of igvm-go package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.igvm-go
      echo "Updating vendorHash of snp-id-block-generator package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.snp-id-block-generator
      echo "Updating vendorHash of contrast.cli package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.contrast

      echo "Updating src hash of kata.kata-kernel-uvm.configfile" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.kata.kata-kernel-uvm.configfile

      echo "Updating yarn offlineCache hash of contrast-docs package" >&2
      nix-update --version=skip --flake \
        --override-filename=packages/by-name/contrast-docs/package.nix \
        legacyPackages.x86_64-linux.contrast-docs

      echo "Updating default kata-container configuration toml files" >&2
      nix run .#scripts.update-kata-configurations
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

      tags="${lib.concatStringsSep "," pkgs.contrast.tags}"
      while IFS= read -r dir; do
        echo "Running govulncheck -tags $tags on $dir"
        govulncheck -C "$dir" -tags "$tags" ./... || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      exit $exitcode
    '';
  };

  gofix = writeShellApplication {
    name = "gofix";
    runtimeInputs = with pkgs; [
      go
      gopls
    ];
    text = ''
      exitcode=0

      tags="${lib.concatStringsSep "," pkgs.contrast.tags}"
      while IFS= read -r dir; do
        echo "Running go fix -tags $tags on $dir"
        go fix -C "$dir" -tags "$tags" ./... || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      # TODO(katexochen): modernize does not support tags?
      while IFS= read -r dir; do
        echo "Running modernize on $dir"
        (cd "$dir" && modernize -fix ./...) || exitcode=$?
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

      tags="${lib.concatStringsSep "," pkgs.contrast.tags}"
      while IFS= read -r dir; do
        echo "Running golangci-lint with tags $tags on $dir" >&2
        golangci-lint run --build-tags "$tags" "$dir/..." || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      echo "Verifying golangci-lint config" >&2
      golangci-lint config verify || exitcode=$?

      exit $exitcode
    '';
  };

  go-licenses-check = writeShellApplication {
    name = "go-licenses-check";
    runtimeInputs = with pkgs; [
      go
      go-licenses
    ];
    text = ''
      exitcode=0

      tags="${lib.concatStringsSep "," pkgs.contrast.tags}"
      while IFS= read -r dir; do
        echo "Downloading Go dependencies for license check" >&2
        go mod -C "$dir" download
        echo "Running go-licenses with tags $tags on $dir" >&2
        GOFLAGS="-tags=$tags" go-licenses check \
          --ignore github.com/edgelesssys/contrast \
          --disallowed_types=restricted,reciprocal,forbidden,unknown \
          "$dir/..." || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

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
      while [[ $timeout -gt 0 ]]; do
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

  kubectl-wait-coordinator = writeShellApplication {
    name = "kubectl-wait-coordinator";
    runtimeInputs = with pkgs; [ kubectl ];
    text = ''
      namespace=$1
      name="coordinator-0"

      echo "Waiting for $name.$namespace to become ready" >&2

      timeout=180

      interval=4
      while [[ $timeout -gt 0 ]]; do
        if kubectl -n "$namespace" get pods "$name"; then
          break
        fi
        sleep "$interval"
        timeout=$((timeout - interval))
      done

      kubectl wait \
         --namespace "$namespace" \
         --for=jsonpath='{.status.phase}'=Running \
         --timeout="''${timeout}s" \
         pod/$name
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
      tag='[a-zA-Z0-9_.-]+'
      sha='@sha256:[a-fA-F0-9]{64}'

      while IFS= read -r replacement; do
        # Get the base name (no tag/sha) and escape dots.
        image_source=$(echo "$replacement" | sed "s/:.*//" | sed "s/\./\\\./g")

        # Get the pinned image that we want to insert, including tag and sha.
        image_target=$(echo "$replacement" | cut -d"=" -f2)

        # expr matches the images we want to replace.
        expr="$image_source:$tag($sha)?"

        # Run replace over all files.
        find "./docs/versioned_docs/version-$MAJOR_MINOR" -type f -exec sed -i -r "s#$expr#$image_target#g" {} \;
      done <"../image-replacements.txt"

      # Replace release artifact download links with the versioned ones.
      repo_url='github\.com/edgelesssys/contrast/releases'
      link_source="$repo_url/(latest/download|download/$tag)/"
      link_target="$repo_url/download/$VERSION/"
      find "./docs/versioned_docs/version-$MAJOR_MINOR" -type f -exec sed -i -r "s#$link_source#$link_target#g" {} \;
    '';
  };

  # Temporary workaround until the sync server is stateful.
  renew-sync-fifo = writeShellApplication {
    name = "renew-sync-fifo";
    runtimeInputs = with pkgs; [ kubectl ];
    text = ''
      kubectl delete configmap -n default sync-server-fifo || true
      syncIP=$(kubectl get svc sync -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
      fifoUUID=$(curl -fsSL "$syncIP:8080/fifo/new" | jq -r '.uuid')
      kubectl create configmap -n default sync-server-fifo --from-literal=uuid="$fifoUUID"
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
          policy=$(echo "$line" | cut -d' ' -f3)
          if [[ -z "$policy" ]]; then
              continue
          fi
          echo "Extracting policy for $namespace.$name" >&2
          echo "$policy" | base64 -d > "$namespace.$name.rego"
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

      retry() {
          local retries=5
          local count=0
          local delay=5
          until "$@"; do
              exit_code=$?
              count=$((count + 1))
              if [ "$count" -lt "$retries" ]; then
                  echo "Command failed. Attempt $count/$retries. Retrying in $delay seconds..."
                  sleep $delay
              else
                  echo "Command failed after $retries attempts. Exiting."
                  return $exit_code
              fi
          done
      }

      if [[ $# -lt 2 ]]; then
        echo "Usage: get-logs [start | download] namespaceFile"
        exit 1
      fi

      case $1 in
      start)
        while ! [[ -s "$2" ]]; do
          sleep 1
        done
        # Check if namespace file exists
        # Since no file exists if no test is run, exit gracefully
        if [[ ! -f "$2" ]]; then
          echo "Namespace file $2 does not exist" >&2
          exit 0
        fi
        namespace="$(head -n1 "$2")"
        cp ./packages/log-collector.yaml ./workspace/log-collector.yaml
        sed -i "s/@@NAMESPACE@@/''${namespace}/g" ./workspace/log-collector.yaml
        retry kubectl apply -f ./workspace/log-collector.yaml 1>/dev/null 2>/dev/null
        ;;
      download)
        if [[ ! -f "$2" ]]; then
          echo "Namespace file $2 does not exist" >&2
          exit 0
        fi
        namespace="$(head -n1 "$2")"
        pod="$(kubectl get pods -o name -n "$namespace" | grep log-collector | cut -c 5-)"
        mkdir -p ./workspace/logs
        retry kubectl wait --for=condition=Ready -n "$namespace" "pod/$pod"
        retry kubectl exec -n "$namespace" "$pod" -- /bin/bash -c "rm -f /exported-logs.tar.gz; cp -r /export /export-no-stream; tar zcvf /exported-logs.tar.gz /export-no-stream; rm -rf /export-no-stream"
        retry kubectl cp -n "$namespace" "$pod:/exported-logs.tar.gz" ./workspace/logs/exported-logs.tar.gz
        tar xzvf ./workspace/logs/exported-logs.tar.gz --directory ./workspace/logs
        ;;
      *)
        echo "Unknown option $1"
        echo "Usage: get-logs [start | download] namespaceFile"
        exit 1
      esac
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

  lint-no-debug = writeShellApplication {
    name = "lint-no-debug";
    runtimeInputs = with pkgs; [ gnugrep ];
    text = ''
      exitcode=0
      for file in "$@"; do
        if grep -i -q -E 'debug.* \? true,' "$file"; then
          echo "Found enabled debug option in $file" >&2
          exitcode=1
        fi
      done
      exit $exitcode
    '';
  };

  cleanup-namespaces = writeShellApplication {
    name = "cleanup-namespaces";
    runtimeInputs = with pkgs; [
      kubectl
    ];
    text = builtins.readFile ./cleanup-namespaces.sh;
  };

  cleanup-containerd = writeShellApplication {
    name = "cleanup-containerd";
    runtimeInputs = with pkgs; [ containerd ];
    text = ''
      while read -r image; do
        ctr --address /run/k3s/containerd/containerd.sock --namespace k8s.io image rm --sync "$image"
      done < <(
        ctr --address /run/k3s/containerd/containerd.sock --namespace k8s.io image list |
          tail -n +2 |
          cut -d' ' -f1
      )
      ctr --address /run/k3s/containerd/containerd.sock --namespace k8s.io content prune references
    '';
  };

  update-kata-configurations = writeShellApplication {
    name = "update-kata-configurations";
    runtimeInputs = with pkgs; [
      yq
      diffutils
    ];
    text = # bash
      ''
        old_defaults="$(git rev-parse --show-toplevel)/nodeinstaller/internal/kataconfig"
        new_defaults="${pkgs.kata.release-tarball}/opt/kata/share/defaults/kata-containers"

        declare -A PLATFORMS=(
          ["clh"]="clh-snp"
          ["qemu-snp"]="qemu-snp"
          ["qemu-tdx"]="qemu-tdx"
        )

        exit_code=0
        for upstream_name in "''${!PLATFORMS[@]}"; do
          platform="''${PLATFORMS[$upstream_name]}"
          old_file="$old_defaults/configuration-$platform.toml"
          new_file="$new_defaults/configuration-$upstream_name.toml"

          if [[ ! -f "$new_file" ]]; then
            # platform has been removed or renamed upstream
            echo "✖ No config for $upstream_name available in upstream source."
            exit_code=1
            continue
          fi

          diff=$(diff "$old_file" "$new_file" || true)
          if [[ -n "$diff" ]]; then
            cp -f "$new_file" "$old_file"
            echo "⚠ Updated config for platform $platform."
          else
            echo "✔ No upstream changes for platform $platform."
          fi
        done
        exit $exit_code
      '';
  };

  nix-gc = writeShellApplication {
    name = "nix-gc";
    runtimeInputs = with pkgs; [ busybox ];
    text = ''
      total=$(df /host/nix | tail -1 | awk '{print $2}')
      avail=$(df /host/nix | tail -1 | awk '{print $4}')
      over=$((total * 1 / 4 - avail)) # Keep 25% of the disk space free
      over=$((over < 0 ? 0 : over * 1024)) # Convert to bytes
      echo "Running nix garbage collection, deleting $over Bytes of store paths"
      nsenter --target 1 --mount -- /root/.nix-profile/bin/nix store gc --max "$over"
    '';
  };

  get-nvidia-rim-ids = writeShellApplication {
    name = "get-nvidia-rim-ids";
    runtimeInputs = with pkgs; [
      curl
      jq
    ];
    text = ''
      curl -fsSL https://rim.attestation.nvidia.com/v1/rim/ids | jq 'del(.request_id)'
    '';
  };
}
