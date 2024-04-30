# Undeploy, rebuild, deploy.
default target=default_deploy_target cli=default_cli: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy node-installer runtime (apply "runtime") (deploy target cli) set verify (wait-for-workload target)

# Build and push a container image.
push target:
    mkdir -p {{ workspace_dir }}
    printf "ghcr.io/edgelesssys/contrast/%s:latest=" "{{ target }}" >> {{ workspace_dir }}/just.containerlookup
    nix run .#containers.push-{{ target }} -- "$container_registry/contrast/{{ target }}" >> {{ workspace_dir }}/just.containerlookup

# Build the coordinator, containerize and push it.
coordinator: (push "coordinator")

# Build the openssl container and push it.
openssl: (push "openssl")

# Build the port-forwarder container and push it.
port-forwarder: (push "port-forwarder")

service-mesh-proxy: (push "service-mesh-proxy")

# Build the initializer, containerize and push it.
initializer: (push "initializer")

# Build the node-installer, containerize and push it.
node-installer: (push "node-installer")

default_cli := "contrast.cli"
default_deploy_target := "openssl"
workspace_dir := "workspace"

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target cli=default_cli: (populate target) (generate cli) (apply target)

# Populate the workspace with a runtime class deployment
runtime:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}/runtime
    nix shell .#contrast --command resourcegen \
      --image-replacements ./{{ workspace_dir }}/just.containerlookup --namespace kube-system \
      runtime > ./{{ workspace_dir }}/runtime/runtime.yml

# Populate the workspace with a Kubernetes deployment
populate target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}
    case {{ target }} in
        "openssl" | "emojivoto")
            mkdir -p ./{{ workspace_dir }}/deployment
            nix shell .#contrast --command resourcegen \
              --image-replacements ./{{ workspace_dir }}/just.containerlookup --namespace {{ target }}${namespace_suffix-} \
              --add-namespace-object --add-port-forwarders \
              {{ target }} coordinator > ./{{ workspace_dir }}/deployment/deployment.yml
        ;;
        *)
            cp -R ./deployments/{{ target }} ./{{ workspace_dir }}/deployment
            nix run .#scripts.patch-contrast-image-hashes -- ./{{ workspace_dir }}/deployment
            nix run .#kypatch images -- ./{{ workspace_dir }}/deployment \
                --replace ghcr.io/edgelesssys ${container_registry}
            nix run .#kypatch namespace -- ./{{ workspace_dir }}/deployment \
                --replace edg-default {{ target }}${namespace_suffix-}
        ;;
    esac
    echo "{{ target }}${namespace_suffix-}" > ./{{ workspace_dir }}/just.namespace

# Generate policies, update manifest.
generate cli=default_cli:
    #!/usr/bin/env bash
    t=$(date +%s)
    nix run .#{{ cli }} -- generate \
        --workspace-dir ./{{ workspace_dir }} \
        ./{{ workspace_dir }}/deployment/*.yml
    duration=$(( $(date +%s) - $t ))
    echo "Generated policies in $duration seconds."
    echo "generate $duration" >> ./{{ workspace_dir }}/just.perf

# Apply Kubernetes manifests from /deployment
apply target=default_deploy_target:
    #!/usr/bin/env bash
    case {{ target }} in
        "runtime")
            kubectl apply -f ./{{ workspace_dir }}/runtime
            exit 0
        ;;
        "openssl" | "emojivoto")
            :
        ;;
        *)
            kubectl apply -f ./{{ workspace_dir }}/deployment/ns.yml
        ;;
    esac
    kubectl apply -f ./{{ workspace_dir }}/deployment

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
    if kubectl get ns $ns 2> /dev/null; then
        kubectl delete \
            -f ./{{ workspace_dir }}/deployment \
            --ignore-not-found \
            --grace-period=30 \
            --timeout=10m
    fi

# Create a CoCo-enabled AKS cluster.
create:
    nix run .#scripts.create-coco-aks -- --name="$azure_resource_group"

# Set the manifest at the coordinator.
set cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run .#scripts.kubectl-wait-ready -- $ns coordinator
    nix run .#scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run .#scripts.wait-for-port-listen -- 1313
    policy=$(< ./{{ workspace_dir }}/coordinator-policy.sha256)
    t=$(date +%s)
    nix run .#{{ cli }} -- set \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313 \
        --coordinator-policy-hash "${policy}" \
        ./{{ workspace_dir }}/deployment/*.yml
    duration=$(( $(date +%s) - $t ))
    echo "Set manifest in $duration seconds."
    echo "set $duration" >> ./{{ workspace_dir }}/just.perf

# Verify the Coordinator.
verify cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{ workspace_dir }}/verify
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run .#scripts.kubectl-wait-ready -- $ns coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1314:1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run .#scripts.wait-for-port-listen -- 1314
    policy=$(< ./{{ workspace_dir }}/coordinator-policy.sha256)
    t=$(date +%s)
    nix run .#{{ cli }} -- verify \
        --workspace-dir ./{{ workspace_dir }}/verify \
        --coordinator-policy-hash "${policy}" \
        -c localhost:1314
    duration=$(( $(date +%s) - $t ))
    echo "Verified in $duration seconds."
    echo "verify $duration" >> ./{{ workspace_dir }}/just.perf

# Wait for workloads to become ready.
wait-for-workload target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    case {{ target }} in
        "openssl")
            nix run .#scripts.kubectl-wait-ready -- $ns openssl-backend
            nix run .#scripts.kubectl-wait-ready -- $ns openssl-client
            nix run .#scripts.kubectl-wait-ready -- $ns openssl-frontend
        ;;
        "emojivoto" | "emojivoto-sm-egress" | "emojivoto-sm-ingress")
            nix run .#scripts.kubectl-wait-ready -- $ns emoji-svc
            nix run .#scripts.kubectl-wait-ready -- $ns vote-bot
            nix run .#scripts.kubectl-wait-ready -- $ns voting-svc
            nix run .#scripts.kubectl-wait-ready -- $ns web-svc
        ;;
        *)
            echo "Please register workloads of new targets in wait-for-workload"
            exit 1
        ;;
    esac

# Load the kubeconfig from the running AKS cluster.
get-credentials:
    nix run .#azure-cli -- aks get-credentials \
        --resource-group "$azure_resource_group" \
        --name "$azure_resource_group"

# Load the kubeconfig from the CI AKS cluster.
get-credentials-ci:
    nix run .#azure-cli -- aks get-credentials \
        --resource-group "contrast-ci" \
        --name "contrast-ci" \
        --admin

# Destroy a running AKS cluster.
destroy:
    nix run .#scripts.destroy-coco-aks -- --name="$azure_resource_group"

# Run code generators.
codegen:
    nix run .#scripts.generate

# Format code.
fmt:
    nix fmt

# Lint code.
lint:
    nix run .#scripts.golangci-lint -- run

demodir namespace="default": coordinator initializer
    #!/usr/bin/env bash
    set -eu
    d=$(mktemp -d)
    echo "Creating demo directory at ${d}" >&2
    nix run .#scripts.fetch-latest-contrast -- {{ namespace }} "${d}"
    unzip -d "${d}" "${d}/emojivoto-demo.zip"
    echo "Demo directory ready at ${d}" >&2
    echo "${d}"

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
# Container registry to push images to
container_registry=""
# Azure resource group/ resource name. Resource group will be created.
azure_resource_group=""

#
# No need to change anything below this line.
#

# Namespace suffix, can be empty. Will be used when patching namespaces.
namespace_suffix=""
# Cache directory for the CLI.
CONTRAST_CACHE_DIR="./workspace.cache"
# Log level for the CLI.
CONTRAST_LOG_LEVEL=""
'''

# Developer onboarding.
onboard:
    @ [[ -f "./justfile.env" ]] && echo "justfile.env already exists" && exit 1 || true
    @echo '{{ rctemplate }}' > ./justfile.env
    @echo "Created ./justfile.env. Please fill it out."

# Just configuration.

set dotenv-filename := "justfile.env"
set dotenv-load := true
set shell := ["bash", "-uc"]
