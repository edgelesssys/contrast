# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  contrastPkgs,
  writeShellApplication,
  runtimePkgs,
}:

lib.makeScope pkgs.newScope (scripts: {
  generate = writeShellApplication {
    name = "generate";
    runtimeInputs = with pkgs; [
      go
      protobuf
      protoc-gen-go
      protoc-gen-go-grpc
      protoc-gen-go-ttrpc
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
      echo "Updating vendorHash of debugshell package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.debugshell
      echo "Updating vendorHash of service-mesh package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.service-mesh
      echo "Updating vendorHash of igvm-go package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.igvm-go
      echo "Updating vendorHash of snp-id-block-generator package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.snp-id-block-generator
      echo "Updating imagepuller package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.imagepuller
      echo "Updating imagestore package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.imagestore
      echo "Updating vendorHash of initdata-processor package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.initdata-processor
      echo "Updating vendorHash of contrast package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.contrast.contrast
      echo "Updating vendorHash of imagepuller-benchmark package" >&2
      nix-update --version=skip --flake legacyPackages.x86_64-linux.imagepuller-benchmark
      echo "Updating src hash of kata.release-tarball" >&2
      ./packages/by-name/kata/release-tarball/update.sh

      echo "Updating yarn offlineCache hash of contrast.docs package" >&2
      nix-update --version=skip --flake \
        --override-filename=packages/by-name/contrast/docs/package.nix \
        legacyPackages.x86_64-linux.contrast.docs

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

      tags="${lib.concatStringsSep "," contrastPkgs.contrast.contrast.tags}"
      while IFS= read -r dir; do
        echo "Running govulncheck -tags $tags on $dir"
        CGO_ENABLED=0 govulncheck -C "$dir" -tags "$tags" ./... || exitcode=$?
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

      tags="${lib.concatStringsSep "," contrastPkgs.contrast.contrast.tags}"
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

      tags="${lib.concatStringsSep "," contrastPkgs.contrast.contrast.tags}"
      while IFS= read -r dir; do
        echo "Running golangci-lint with tags $tags on $dir" >&2
        CGO_ENABLED=0 golangci-lint run --build-tags "$tags" "$dir/..." || exitcode=$?
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

      ignoreFlags=(
        --ignore github.com/edgelesssys/contrast
        --ignore github.com/rootless-containers/proto/go-proto # Apache-2.0, but not correctly recognized https://github.com/rootless-containers/proto/issues/5
        --ignore github.com/cyphar/filepath-securejoin # MPL-2.0, which is reciprocal but only for the original source.
      )

      tags="${lib.concatStringsSep "," contrastPkgs.contrast.contrast.tags}"
      while IFS= read -r dir; do
        echo "Downloading Go dependencies for license check" >&2
        go mod -C "$dir" download
        echo "Running go-licenses with tags $tags on $dir" >&2
        GOFLAGS="-tags=$tags" go-licenses check \
          "''${ignoreFlags[@]}" \
          --disallowed_types=restricted,reciprocal,forbidden,unknown \
          "$dir/..." || exitcode=$?
      done < <(go list -f '{{.Dir}}' -m)

      exit $exitcode
    '';
  };

  # Use: go-closure ./foo/... ./bar/...
  go-closure = writeShellApplication {
    name = "go-closure";
    runtimeInputs = with pkgs; [ go ];
    text = ''
      tags="${lib.concatStringsSep "," contrastPkgs.contrast.e2e.tags}"
      go list -tags "$tags" -deps -f '{{ if ne .Module nil }}{{ if .Module.Main }}{{ .ImportPath }}{{ end }}{{ end }}' "$@" | sort
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

  wait-for-port-listen =
    let
      inherit (pkgs.stdenv.hostPlatform) isDarwin;
    in
    writeShellApplication {
      name = "wait-for-port-listen";
      runtimeInputs = if isDarwin then [ pkgs.lsof ] else [ pkgs.iproute2 ];
      text = ''
          port=$1
          tries=15 # 3 seconds
          interval=0.2

          function listen-on-port() {
            ${
              if isDarwin then
                ''
                  lsof -i :"$port" -sTCP:LISTEN -t
                ''
              else
                ''
                  ss --tcp --numeric --listening --no-header --ipv4 src ":$port"
                ''
            }
          }

          while [[ "$tries" -gt 0 ]]; do
            if [[ -n $(listen-on-port) ]]; then
              exit 0
            fi
            sleep "$interval"
            tries=$((tries - 1))
          done

        echo "Port $port did not reach state LISTENING" >&2
        exit 1
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

  renew-sync-fifo = writeShellApplication {
    name = "renew-sync-fifo";
    runtimeInputs = with pkgs; [ kubectl ];
    text = ''
      kubectl delete configmap -n default sync-server-fifo || true
      syncIP=$(kubectl get svc sync -n default -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
      fifoUUID=$(curl -fsSL "$syncIP:8080/fifo/new?allow_overrides=true" | jq -r '.uuid')
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
      sed 's| /run/measured-cfg/policy.rego:|\nagent_policy:|g' |
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
          echo "$policy" | base64 -d | gunzip > "$namespace.$name.initdata.toml"
          yq --input-format toml '.data["policy.rego"]' < "$namespace.$name.initdata.toml" > "$namespace.$name.policy.rego"
      done < <(
        yq '.metadata.name
          + " " + (
            .metadata.namespace
            // "default"
          ) + " " + (
            .spec.template.metadata.annotations["io.katacontainers.config.hypervisor.cc_init_data"]
            // .metadata.annotations["io.katacontainers.config.hypervisor.cc_init_data"]
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
    name = "get-credentials";
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

  # Usage: get-ghcr-read-token $gcloudSecretRef
  get-ghcr-read-token = writeShellApplication {
    name = "get-ghcr-read-token";
    runtimeInputs = with pkgs; [
      google-cloud-sdk
    ];
    text = ''
      set -euo pipefail
      gcloud secrets versions access "$1"
    '';
  };

  cleanup-bare-metal = runtimePkgs.writeShellApplication {
    name = "cleanup-bare-metal";
    runtimeInputs = with runtimePkgs; [
      busybox
      kubectl
      dasel
      scripts.cleanup-images
    ];
    text = builtins.readFile ./cleanup-bare-metal.sh;
  };

  cleanup-images = runtimePkgs.writeShellApplication {
    name = "cleanup-images";
    runtimeInputs = with runtimePkgs; [
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
        ver=$(grep -E '^go [0-9]+[.][0-9]+([.][0-9]+)?$' "$f")
        goVers+=("$ver")
      done <<< "$modFiles"

      maxVer=$(printf "%s\n" "''${goVers[@]}" | sort -V | tail -n1)

      while IFS= read -r f; do
        sed -i "s/^go [0-9]\+\.[0-9]\+\(\.[0-9]\+\)\?$/''${maxVer}/" "$f"
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
      curl
    ];
    text = builtins.readFile ./cleanup-namespaces.sh;
  };

  cleanup-containerd = runtimePkgs.writeShellApplication {
    name = "cleanup-containerd";
    runtimeInputs = with runtimePkgs; [ containerd ];
    text = ''
      declare address
      if [[ -S "/host/run/k3s/containerd/containerd.sock" ]]; then
        address="/host/run/k3s/containerd/containerd.sock"
      elif [[ -S "/host/run/containerd/containerd.sock" ]]; then
        address="/host/run/containerd/containerd.sock"
      else
        echo "No containerd socket found at /run/containerd/containerd.sock or /run/k3s/containerd/containerd.sock"
        exit 1
      fi
      while read -r image; do
        ctr --address "$address" --namespace k8s.io image rm --sync "$image"
      done < <(
        ctr --address "$address" --namespace k8s.io image list |
          tail -n +2 |
          cut -d' ' -f1
      )
      ctr --address "$address" --namespace k8s.io content prune references
    '';
  };

  update-kata-configurations = writeShellApplication {
    name = "update-kata-configurations";
    runtimeInputs = [
      (pkgs.buildGoModule {
        inherit (contrastPkgs.contrast.contrast) vendorHash;
        name = "nodeinstaller-kataconfig-update-testdata";

        src =
          let
            inherit (lib) fileset path hasSuffix;
            root = ../.;
          in
          lib.fileset.toSource {
            inherit root;
            fileset = fileset.unions [
              (path.append root "go.mod")
              (path.append root "go.sum")
              (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/platforms"))
              (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "nodeinstaller"))
              (fileset.fileFilter (file: hasSuffix ".toml" file.name) (path.append root "nodeinstaller"))
              (fileset.fileFilter (file: hasSuffix ".json" file.name) (path.append root "nodeinstaller"))
            ];
          };

        proxyVendor = true;
        subPackages = [ "nodeinstaller/internal/kataconfig/update-testdata" ];

        env.CGO_ENABLED = 0;
        ldflags = [ "-s" ];
        doCheck = false;
      })
      pkgs.git
    ];
    text = # bash
      ''
        update-testdata ${contrastPkgs.kata.release-tarball} "$(git rev-parse --show-toplevel)"
      '';
  };

  nix-gc = runtimePkgs.writeShellApplication {
    name = "nix-gc";
    runtimeInputs = with runtimePkgs; [ busybox ];
    text = ''
      total=$(df -P /host/nix | tail -1 | awk '{print $2}')
      avail=$(df -P /host/nix | tail -1 | awk '{print $4}')
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

  get-nvidia-cc-gpus = writeShellApplication {
    name = "get-nvidia-cc-gpus";
    runtimeInputs = with pkgs; [
      curl
      jq
    ];
    text = ''
      readonly url="https://www.nvidia.com/content/dam/en-zz/Solutions/data-center/solutions/confidential-computing/compatibility-matrix/secure-ai-compatibility-matrix-v0.01.js"

      driverVer="$1"
      export driverVer

      curl -fsSL  "$url" \
        | sed -n '/const matrix = \[/,/];/p' \
        | sed 's/const matrix = //' \
        | sed 's/];/{}]/' \
        | jq -r --arg driverVer "$driverVer" '.[]
          | select(."CUDA Driver" == $driverVer)
          | select(."Confidential Computing Mode" == "Single GPU Passthrough")
          | select(."RIM Status" == "Released")
          | .Description
        ' \
        | sort -u
    '';
  };

  get-sync-ticket = writeShellApplication {
    name = "get-sync-ticket";
    runtimeInputs = with pkgs; [
      kubectl
      jq
      curl
    ];
    text = ''
      echo "Requesting fifo ticket from sync server" >&2
      sync_ip=$(kubectl get svc sync -n default -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
      echo "Sync server IP: $sync_ip" >&2
      sync_uuid=$(kubectl get configmap sync-server-fifo -n default -o jsonpath='{.data.uuid}')
      echo "Sync fifo UUID: $sync_uuid" >&2
      path="ticket"
      if [[ "$#" -ge 1 ]]; then
        path="ticket?done_timeout=$1"
      fi
      sync_ticket=$(curl -fsSL "$sync_ip:8080/fifo/$sync_uuid/$path" | jq -r '.ticket')
      echo "Waiting for lock on fifo $sync_uuid with ticket $sync_ticket" >&2
      curl -fsSL "$sync_ip:8080/fifo/$sync_uuid/wait/$sync_ticket"
      echo "Acquired lock on fifo $sync_uuid with ticket $sync_ticket" >&2
      echo "$sync_ticket"
    '';
  };

  release-sync-ticket = writeShellApplication {
    name = "get-sync-ticket";
    runtimeInputs = with pkgs; [
      kubectl
      curl
    ];
    text = ''
      ticket=$1
      echo "Releasing fifo ticket $ticket" >&2
      sync_ip=$(kubectl get svc sync -n default -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
      echo "Sync server IP: $sync_ip" >&2
      sync_uuid=$(kubectl get configmap sync-server-fifo -n default -o jsonpath='{.data.uuid}')
      echo "Sync fifo UUID: $sync_uuid" >&2
      if ! curl -fsSL "$sync_ip:8080/fifo/$sync_uuid/done/$ticket"; then
        echo "Failed to release fifo $sync_uuid with ticket $ticket" >&2
      else
        echo "Successfully released fifo $sync_uuid with ticket $ticket" >&2
      fi
    '';
  };

  check-sidebar = writeShellApplication {
    name = "check-sidebar";
    runtimeInputs = [ pkgs.nodejs ];
    text = ''
      node ${./check-sidebar.cjs} "$@"
    '';
  };

  # ascii-lint ensures that documentation files are restricted to the ASCII character set.
  ascii-lint = writeShellApplication {
    name = "ascii-lint";
    runtimeInputs = [ pkgs.gnugrep ];
    text = ''
      # The range expression below is locale-dependent, we want it to work in standard ASCII.
      export LC_ALL="C"
      # Printable ASCII range is 0x20 SPACE to 0x7E TILDE.
      if grep -n '[^ -~]' "$@"; then
        echo "Found non-ASCII characters in above files. Is this an AI generated text?" >&2
        exit 1
      fi
    '';
  };

  docs-selfref-lint = writeShellApplication {
    name = "docs-selfref-lint";
    runtimeInputs = with pkgs; [ gnugrep ];
    text = ''
      echo "Checking for internal documentation references to docs.edgeless.systems/contrast" >&2
      if grep -lF 'docs.edgeless.systems/contrast' "$@"; then
        echo "Found a references to docs.edgeless.systems/contrast" >&2
        echo "Please use relative links for internal documentation references." >&2
        exit 1
      fi
    '';
  };

  # Use after adding a new image to the imagepuller-benchmark, to write the
  # maximum allowed storage usage to the benchmark file.
  imagepuller-benchmark-update-sizes = writeShellApplication {
    name = "imagepuller-benchmark-update-sizes";
    runtimeInputs = with pkgs; [
      docker
      jq
    ];
    text = ''
      TOLERANCE=120 # storage must not exceed 120% of the uncompressed image size
      total_size=0

      INPUT_FILE="./tools/imagepuller-benchmark/benchmark.json"
      TMP_FILE="$(mktemp)"
      cp "$INPUT_FILE" "$TMP_FILE"

      keys=$(jq -r 'keys[]' "$TMP_FILE")
      for key in $keys; do
          has_image=$(jq -r --arg k "$key" '.[$k] | has("image")' "$TMP_FILE")
          if [[ "$has_image" == "true" ]]; then
              image=$(jq -r --arg k "$key" '.[$k].image' "$TMP_FILE")
              docker pull "$image" > /dev/null
              size=$(docker inspect -f "{{ .Size }}" "$image")
              size_mb=$(( size / 1024 / 1024 ))
              size_mb_with_margin=$(( (size_mb * TOLERANCE) / 100 ))
              total_size=$(( total_size + size_mb_with_margin ))
              jq --arg k "$key" --argjson s "$size_mb_with_margin" \
                 '.[$k].storage = $s' \
                 "$TMP_FILE" > "''${TMP_FILE}.new"
              mv "''${TMP_FILE}.new" "$TMP_FILE"
          fi
      done

      jq --argjson total "$total_size" \
        '.continuous.storage = $total' \
        "$TMP_FILE" > "''${TMP_FILE}.new"

      mv "$TMP_FILE.new" "$INPUT_FILE"
    '';
  };

  # This is a script rather than part of packages/containers.nix because we *want* impurity here,
  # in order to generate a new image digest and a new tag every time the script runs.
  push-containerd-reproducer = writeShellApplication {
    name = "push-containerd-reproducer";
    runtimeInputs = with pkgs; [
      jq
      umoci
      skopeo
    ];
    text = ''
      registry="$1"
      tmpdir="$(mktemp -d)/oci"
      timestamp=$(date +%s)

      umoci init --layout "$tmpdir"
      skopeo copy "docker://ghcr.io/edgelesssys/bash@sha256:cabc70d68e38584052cff2c271748a0506b47069ebbd3d26096478524e9b270b" "oci:$tmpdir:alpine" --insecure-policy
      umoci unpack --image "$tmpdir:alpine" "$tmpdir/rootfs" --rootless
      echo "$timestamp" > "$tmpdir/rootfs/rootfs/timestamp"
      umoci repack --image "$tmpdir:alpine" "$tmpdir/rootfs"
      skopeo copy "oci:$tmpdir" "docker://$registry/contrast/containerd-reproducer:$timestamp" --insecure-policy

      digest=$(jq -r '.manifests[0].digest' "$tmpdir/index.json")
      echo "$timestamp $digest"
    '';
  };

  upgrade-gpu-operator = runtimePkgs.writeShellApplication {
    name = "upgrade-gpu-operator";
    runtimeInputs = with runtimePkgs; [
      busybox
      kubectl
      kubernetes-helm
    ];
    text = builtins.readFile ./upgrade-gpu-operator.sh;
  };
})
