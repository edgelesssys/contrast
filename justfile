default_cli := "contrast.cli"
default_deploy_target := "openssl"
default_platform := "${default_platform}"
default_set := "${set}"
workspace_dir := "workspace"

# Undeploy, rebuild, deploy.
default target=default_deploy_target platform=default_platform cli=default_cli: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump debugshell (deploy target cli platform) set verify (wait-for-workload target)

# Build and push a container image.
push target set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ workspace_dir }}
    echo "Pushing container $container_registry/contrast/{{ target }}"
    nix run -L .#{{ set }}.scripts.containers.push-{{ target }} -- "$container_registry/contrast/{{ target }}" "{{ workspace_dir }}/just.containerlookup" "{{ workspace_dir }}/layers-cache.json"

coordinator: (push "coordinator")

openssl: (push "openssl")

port-forwarder: (push "port-forwarder")

service-mesh-proxy: (push "service-mesh-proxy")

initializer: (push "initializer")

memdump: (push "memdump")

debugshell: (push "debugshell")

k8s-log-collector: (push "k8s-log-collector")

# Download all logs (pod logs + host journal). Deploys the log-collector if not already running.
download-logs set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    # Only push if not already pushed (e.g. by _e2e).
    if ! grep -q "k8s-log-collector" "{{ workspace_dir }}/just.containerlookup" 2>/dev/null; then
      just k8s-log-collector
    fi
    namespace_file="{{ workspace_dir }}/just.namespace"
    if [[ ! -f "$namespace_file" ]]; then
      echo "No namespace file found at $namespace_file. Deploy something first." >&2
      exit 1
    fi
    nix run .#{{ set }}.scripts.get-logs -- download "$namespace_file"

containerd-reproducer set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    read tag digest < <(nix run -L .#{{ set }}.scripts.containers.push-containerd-reproducer -- $container_registry | tail -n 1)
    echo "ghcr.io/edgelesssys/contrast/containerd-reproducer:latest-tag=$container_registry/contrast/containerd-reproducer:$tag" >> {{ workspace_dir }}/just.containerlookup
    echo "ghcr.io/edgelesssys/contrast/containerd-reproducer:latest-digest=$container_registry/contrast/containerd-reproducer@$digest" >> {{ workspace_dir }}/just.containerlookup

# Build the node-installer, containerize and push it.
node-installer platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "Metal-QEMU-SNP"|"Metal-QEMU-TDX")
            just push "node-installer-kata"
        ;;
        "Metal-QEMU-SNP-GPU"|"Metal-QEMU-TDX-GPU")
            just push "node-installer-kata-gpu"
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Get all used platform names in a comma-separated string
collect-platforms platform=default_platform set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    deployment=$( [[ -e "./{{ workspace_dir }}/deployment" ]] && printf '%s' "./{{ workspace_dir }}/deployment" || printf '' )
    nix shell .#{{ set }}.contrast.resourcegen --command resourcegen \
        --platform {{ platform }} \
        --deployment "$deployment" \
        collect-platforms

# Some e2e tests require a specific Nix package set to build correctly. Auto-select the correct set based on the test name.
e2e target=default_deploy_target platform=default_platform set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    declare -A required_sets=(
        [badaml-vuln]=badaml-vuln
        [badaml-sandbox]=badaml-sandbox
    )
    RESOLVED_SET="{{ set }}"
    if [[ -v "required_sets[{{ target }}]" ]]; then
        RESOLVED_SET="${required_sets[{{ target }}]}"
    fi
    echo "Using set=$RESOLVED_SET for test '{{ target }}'"
    set="$RESOLVED_SET" just _e2e {{ target }} {{ platform }}

_e2e target=default_deploy_target platform=default_platform set=default_set: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump debugshell k8s-log-collector (node-installer platform)
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ {{ platform }} == "Metal-QEMU-SNP-GPU" || {{ platform }} == "Metal-QEMU-TDX-GPU" ]] ; then
        just request-fifo-ticket 90m
    fi
    if [[ -n "${contrast_ghcr_read:-}" ]]; then
        export CONTRAST_GHCR_READ="$contrast_ghcr_read"
    fi
    if [[ {{ target }} == "multi-runtime-class" ]]; then
        # both are required
        just push "node-installer-kata"
        just push "node-installer-kata-gpu"
    fi
    if [[ {{ target }} == "containerd-11644-reproducer" ]]; then
        just containerd-reproducer
    fi
    get_logs=$(nix build .#{{ set }}.scripts.get-logs --no-link --print-out-paths)
    "$get_logs/bin/get-logs" start ./{{ workspace_dir }}/just.namespace &
    get_logs_pid=$!
    trap 'kill $get_logs_pid || true' EXIT
    nix shell .#{{ set }}.contrast.e2e --command {{ target }}.test -test.v \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace-file ./{{ workspace_dir }}/just.namespace \
            --platform {{ platform }} \
            --node-installer-target-conf-type "${node_installer_target_conf_type}" \
            --namespace-suffix=${namespace_suffix-} \
            --sync-ticket-file ./{{ workspace_dir }}/just.sync-ticket \
            --insecure-enable-debug-shell-access=${debug:-false}

e2e-release version platform=default_platform set=default_set: soft-clean
    #!/usr/bin/env bash
    set -euo pipefail
    nix build .#{{ set }}.scripts.get-logs
    mkdir -p ./{{ workspace_dir }}
    nix run .#{{ set }}.scripts.get-logs start ./{{ workspace_dir }}/just.namespace &
    trap "kubectl delete -f ./{{ workspace_dir }}/log-collector.yaml; rm -f ./{{ workspace_dir }}/just.namespace" EXIT
    nix shell .#{{ set }}.contrast.e2e --command release.test -test.v \
            --tag {{ version }} \
            --platform {{ platform }} \
            --node-installer-target-conf ${node_installer_target_conf_type} \
            --namespace-file ./{{ workspace_dir }}/just.namespace \
            --use-loadbalancer=true
    nix run .#{{ set }}.scripts.get-logs download ./{{ workspace_dir }}/just.namespace

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target cli=default_cli platform=default_platform: (populate target platform) (runtime target platform) (write-namespace target) (apply "runtime" platform) (generate cli platform) (apply target platform)

# Populate the workspace with a runtime class deployment
runtime target=default_deploy_target platform=default_platform set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    platforms=$(just collect-platforms)
    IFS=',' read -ra PLATFORMS <<< "$platforms"
    for platform in "${PLATFORMS[@]}"; do
        just node-installer "$platform"
    done

    mkdir -p ./{{ workspace_dir }}/runtime
    if [[ "${node_installer_target_conf_type}" != "none" ]]; then
        nix shell .#{{ set }}.contrast.resourcegen --command resourcegen \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace {{ target }}${namespace_suffix-} \
            --node-installer-target-conf-type ${node_installer_target_conf_type} \
            --platform {{ platform }} \
            node-installer-target-conf > ./{{ workspace_dir }}/runtime/target-conf.yml
    fi

    nix shell .#{{ set }}.contrast.resourcegen --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --node-installer-target-conf-type ${node_installer_target_conf_type} \
        --platform "$platforms" \
        runtime >> "./{{ workspace_dir }}/runtime/runtime.yml"

# Populate the workspace with a Kubernetes deployment
populate target=default_deploy_target platform=default_platform set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}
    mkdir -p ./{{ workspace_dir }}/deployment
    target="{{ target }}"
    if [[ "${target}" == "custom" ]]; then
        target=""
        cp -r ./.custom/* ./{{ workspace_dir }}/deployment/
        ns="{{ target }}${namespace_suffix-}"
        for file in ./{{ workspace_dir }}/deployment/*.yml; do
            env ns="$ns" yq -i '.metadata.namespace = strenv(ns)' "$file"
        done
    fi
    if [[ -f ./{{ workspace_dir }}/deployment/deployment.yml ]]; then
        echo "---" >> ./{{ workspace_dir }}/deployment/deployment.yml
    fi
    dmesgFlag=""
    # For debug, we already add the debugshell container which exposes the full journal.
    if [[ "${debug:-}" != "true" ]]; then
        dmesgFlag="--add-dmesg"
    fi
    gpuFlags=()
    if [[ {{ platform }} == "Metal-QEMU-TDX-GPU" ]] ; then
        gpuFlags=("--gpu-class" "nvidia.com/GB100_B200")
    fi
    storageClassFlags=()
    storageClass=$(kubectl get storageclass -l ci.contrast.edgeless.systems/is-default-class=true -o "jsonpath={.items[*]['metadata.name']}")
    if [[ -n "$storageClass" ]]; then
        storageClassFlags=("--storage-class" "$storageClass")
    fi
    nix shell .#{{ set }}.contrast.resourcegen --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --add-port-forwarders \
        --add-logging \
        ${dmesgFlag} \
        "${gpuFlags[@]}" \
        "${storageClassFlags[@]}" \
        --platform {{ platform }} \
        ${target} coordinator >> ./{{ workspace_dir }}/deployment/deployment.yml

# Write the namespace so it can be read by other scripts
write-namespace target=default_deploy_target:
    echo "{{ target }}${namespace_suffix-}" >> ./{{ workspace_dir }}/just.namespace

# Generate policies, update manifest.
generate cli=default_cli platform=default_platform set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    debugFlag=""
    if [[ "${debug:-}" == "true" ]]; then
        debugFlag="--insecure-enable-debug-shell-access"
    fi
    patch=$(mktemp)
    kubectl -n default get cm bm-tcb-specs -o jsonpath="{.data['specs']}" > "$patch"
    nix run -L .#{{ set }}.{{ cli }} -- generate \
        --workspace-dir ./{{ workspace_dir }} \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --reference-values {{ platform }} \
        --reference-value-patches "$patch" \
        --purge-empty-reference-values \
        ${debugFlag} \
        ./{{ workspace_dir }}/deployment/

# Apply Kubernetes manifests from /deployment
apply target=default_deploy_target platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(tail -1 ./{{ workspace_dir }}/just.namespace)
    kubectl create namespace $ns --dry-run=client -o yaml | kubectl apply -f -
    case {{ target }} in
        "runtime")
            if [[ -f ./{{ workspace_dir }}/runtime/target-conf.yml ]]; then
                kubectl apply -f ./{{ workspace_dir }}/runtime/target-conf.yml
            fi
            if kubectl get secret -n default contrast-e2e-registry-auth >/dev/null 2>&1; then
              kubectl get secret -n default contrast-e2e-registry-auth -o 'jsonpath={.data.contrast-imagepuller\.toml}' \
              | base64 -d \
              | kubectl create secret generic -n "$ns" contrast-node-installer-imagepuller-config --from-file=contrast-imagepuller.toml=/dev/stdin
            elif [[ -n "${contrast_ghcr_read:-}" ]]; then
                cat > "./{{ workspace_dir }}/contrast-imagepuller.toml" <<EOF
    [registries]
    [registries."ghcr.io."]
    auth = "$(printf "user-not-required-here:%s" "$contrast_ghcr_read" | base64 -w0)"
    EOF
                kubectl create secret generic contrast-node-installer-imagepuller-config \
                    --from-file "contrast-imagepuller.toml"="./{{ workspace_dir }}/contrast-imagepuller.toml" \
                    --namespace $(tail -1 ./{{ workspace_dir }}/just.namespace)
            fi
            kubectl apply -f ./{{ workspace_dir }}/runtime/runtime.yml
        ;;
        *)
            if [[ {{ platform }} == "Metal-QEMU-SNP-GPU" || {{ platform }} == "Metal-QEMU-TDX-GPU"  ]] ; then
                just request-fifo-ticket 90m
                trap 'just release-fifo-ticket' ERR
                kubectl label ns $(tail -1 ./{{ workspace_dir }}/just.namespace) contrast.edgeless.systems/sync-ticket=$(cat ./{{ workspace_dir }}/just.sync-ticket) --overwrite
            fi
            kubectl apply -f ./{{ workspace_dir }}/deployment
        ;;
    esac

# Delete Kubernetes manifests.
undeploy:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! -d ./{{ workspace_dir }} ]]; then
        echo "No workspace directory found, nothing to undeploy."
        exit 0
    fi
    if [[ ! -f ./{{ workspace_dir }}/just.namespace ]]; then
        echo "No namespace file found, nothing to undeploy."
        exit 0
    fi
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    kubectl delete namespace $ns --ignore-not-found --timeout 10m
    rm -f ./{{ workspace_dir }}/just.namespace
    just release-fifo-ticket

# Set the manifest at the coordinator.
set cli=default_cli set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(tail -1 ./{{ workspace_dir }}/just.namespace)
    nix run -L .#{{ set }}.scripts.kubectl-wait-coordinator -- $ns
    nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#{{ set }}.scripts.wait-for-port-listen -- 1313
    nix run -L .#{{ set }}.{{ cli }} -- set \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313 \
        ./{{ workspace_dir }}/deployment/

# Verify the Coordinator.
verify cli=default_cli set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{ workspace_dir }}/verify
    ns=$(tail -1 ./{{ workspace_dir }}/just.namespace)
    nix run -L .#{{ set }}.scripts.kubectl-wait-coordinator -- $ns
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1314:1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#{{ set }}.scripts.wait-for-port-listen -- 1314
    nix run -L .#{{ set }}.{{ cli }} -- verify \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1314

# Recover the Coordinator.
recover cli=default_cli set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(tail -1 ./{{ workspace_dir }}/just.namespace)
    nix run -L .#{{ set }}.scripts.kubectl-wait-coordinator -- $ns
    nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#{{ set }}.scripts.wait-for-port-listen -- 1313
    nix run -L .#{{ set }}.{{ cli }} -- recover \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313

# Wait for workloads to become ready.
wait-for-workload target=default_deploy_target set=default_set:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(tail -1 ./{{ workspace_dir }}/just.namespace)
    case {{ target }} in
        "openssl")
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns openssl-backend
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns openssl-frontend
        ;;
        "emojivoto" | "emojivoto-sm-ingress")
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns emoji-svc
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns vote-bot
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns voting-svc
            nix run -L .#{{ set }}.scripts.kubectl-wait-ready -- $ns web-svc
        ;;
        "volume-stateful-set")
            nix run .#{{ set }}.scripts.kubectl-wait-ready -- $ns volume-tester
        ;;
        "mysql")
            nix run .#{{ set }}.scripts.kubectl-wait-ready -- $ns mysql-backend
            nix run .#{{ set }}.scripts.kubectl-wait-ready -- $ns mysql-client
        ;;
        "vault")
            nix run .#{{ set }}.scripts.kubectl-wait-ready -- $ns vault
        ;;
        "gpu")
            nix run .#{{ set }}.scripts.kubectl-wait-ready -- $ns gpu-tester
        ;;
        "custom")
            :
        ;;
        *)
            echo "Please register workloads of new targets in wait-for-workload"
            exit 1
        ;;
    esac

request-fifo-ticket timeout="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ -f ./{{ workspace_dir }}/just.sync-ticket ]]; then
        echo "Sync ticket already exists, not requesting a new one."
        exit 1
    fi
    ticket=$(nix run .#base.scripts.get-sync-ticket {{ timeout }})
    mkdir -p ./{{ workspace_dir }}
    echo $ticket > ./{{ workspace_dir }}/just.sync-ticket

release-fifo-ticket:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! -f ./{{ workspace_dir }}/just.sync-ticket ]]; then
        echo "No sync ticket found, nothing to release."
        exit 0
    fi
    ticket=$(cat ./{{ workspace_dir }}/just.sync-ticket)
    nix run .#base.scripts.release-sync-ticket ${ticket}
    rm ./{{ workspace_dir }}/just.sync-ticket

# Load the kubeconfig for the given platform.
get-credentials platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "Metal-QEMU-TDX"|"olimar")
            nix run -L .#base.scripts.get-credentials "projects/796962942582/secrets/olimar-kubeconfig/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-TDX"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="k3s"/' justfile.env
        ;;
        "Metal-QEMU-TDX-GPU"|"dgx-007")
            nix run -L .#base.scripts.get-credentials "projects/796962942582/secrets/dgx-007-kubeconfig/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-TDX-GPU"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="none"/' justfile.env
        ;;
        "Metal-QEMU-SNP"|"palutena")
            nix run -L .#base.scripts.get-credentials "projects/796962942582/secrets/palutena-kubeconfig/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-SNP"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="none"/' justfile.env
        ;;
        "Metal-QEMU-SNP-GPU"|"discovery")
            nix run -L .#base.scripts.get-credentials "projects/796962942582/secrets/discovery-kubeconf/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-SNP-GPU"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="k3s"/' justfile.env
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Load the kubeconfig from the dev cluster.
get-credentials-dev:
    nix run -L .#base.scripts.get-credentials "projects/796962942582/secrets/hetzner-ax162-snp-kubeconfig/versions/latest"
    sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-SNP"/' justfile.env
    sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="k3s"/' justfile.env

# Get the Github token with read access to Contrast's ghcr.io packages.
get-ghcr-read-token:
    #!/usr/bin/env bash
    set -euo pipefail
    token=$(nix run -L .#base.scripts.get-ghcr-read-token "projects/796962942582/secrets/ghcr-read-token/versions/latest")
    sed -i "s/^contrast_ghcr_read=.*/contrast_ghcr_read=\"${token}\"/" justfile.env

# Run code generators.
codegen:
    nix run -L .#base.scripts.generate

# Format code.
fmt:
    nix fmt

# Lint code.
lint:
    nix run -L .#base.scripts.golangci-lint -- run

# Check links.
check-links:
    nix run .#base.nixpkgs.lychee -- --config tools/lychee/config-external.toml .

demodir version="latest": undeploy
    #!/usr/bin/env bash
    set -euo pipefail
    v="$(echo {{ version }} | sed 's/\./-/g')"
    nix develop -u DIRENV_DIR -u DIRENV_FILE -u DIRENV_DIFF -u DIRENV_WATCHES .#base.demo-$v

# Remove deployment specific files.
soft-clean: undeploy
    rm -rf ./{{ workspace_dir }}

# Cleanup all auxiliary files, caches etc.
clean: soft-clean
    rm -rf ./{{ workspace_dir }}.cache
    rm -rf ./layers_cache
    rm -f ./layers-cache.json

# Template for the justfile.env file.

rctemplate := '''
# Container registry to push images to, i.e ghcr.io/<your username>
container_registry=""
# Platform to deploy on
default_platform="Metal-QEMU-SNP"
# Node installer target config map to deploy.
node_installer_target_conf_type="k3s"
# Enable insecure debug features like debug shell access.
debug="false"
# Set of packages to use.
set="base"
# Namespace suffix, can be empty. Will be used when patching namespaces.
namespace_suffix=""
# Cache directory for the CLI.
CONTRAST_CACHE_DIR="./workspace.cache"
# Log level for the CLI.
CONTRAST_LOG_LEVEL=""
# A Github token with read access to the Contrast ghcr.io packages.
# Should be set by running just get-ghcr-read-token.
contrast_ghcr_read=""
# A Github token with contents:write permissions for accessing draft releases.
# Set this manually if you want to use the e2e-release target on a draft release.
GH_TOKEN=""
'''

# Developer onboarding.
onboard:
    @ [[ -f "./justfile.env" ]] && echo "justfile.env already exists" && exit 1 || true
    @echo '{{ rctemplate }}' > ./justfile.env
    @just get-ghcr-read-token
    @echo "Created ./justfile.env. Please fill it out."

# Just configuration.

set dotenv-filename := "justfile.env"
set dotenv-load := true
set shell := ["bash", "-uc"]
