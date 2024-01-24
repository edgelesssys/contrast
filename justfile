# Undeploy, rebuild, deploy.
default target=default_deploy_target: undeploy coordinator initializer openssl (deploy target) set verify (wait-for-workload target)

# Build the coordinator, containerize and push it.
coordinator:
    nix run .#push-coordinator -- "$container_registry/nunki/coordinator"

# Build the openssl container and push it.
openssl:
    nix run .#push-openssl -- "$container_registry/nunki/openssl"

# Build the initializer, containerize and push it.
initializer:
    nix run .#push-initializer -- "$container_registry/nunki/initializer"

default_deploy_target := "simple"
workspace_dir := "workspace"

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target: (generate target) apply

# Generate policies, update manifest.
generate target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}
    rm -rf ./{{ workspace_dir }}/*
    cp -R ./deployments/{{ target }} ./{{ workspace_dir }}/deployment
    echo "{{ target }}${namespace_suffix-}" > ./{{ workspace_dir }}/just.namespace
    nix run .#patch-nunki-image-hashes -- ./{{ workspace_dir }}/deployment
    nix run .#kypatch images -- ./{{ workspace_dir }}/deployment \
        --replace ghcr.io/edgelesssys ${container_registry}
    nix run .#kypatch namespace -- ./{{ workspace_dir }}/deployment \
        --replace edg-default {{ target }}${namespace_suffix-}
    t=$(date +%s)
    nix run .#cli -- generate \
        -m ./{{ workspace_dir }}/manifest.json \
        -p ./{{ workspace_dir }} \
        -s genpolicy-msft.json \
        ./{{ workspace_dir }}/deployment/*.yml
    duration=$(( $(date +%s) - $t ))
    echo "Generated policies in $duration seconds."
    echo "generate $duration" >> ./{{ workspace_dir }}/just.perf

# Apply Kubernetes manifests from /deployment
apply:
    kubectl apply -f ./{{ workspace_dir }}/deployment/ns.yml
    kubectl apply -f ./{{ workspace_dir }}/deployment

# Delete Kubernetes manifests.
undeploy:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! -d ./{{ workspace_dir }} ]]; then
        echo "No workspace directory found, nothing to undeploy."
        exit 0
    fi
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    if kubectl get ns $ns 2> /dev/null; then
        kubectl delete -f ./{{ workspace_dir }}/deployment
    fi

# Create a CoCo-enabled AKS cluster.
create:
    nix run .#create-coco-aks -- --name="$azure_resource_group"

# Set the manifest at the coordinator.
set:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run .#kubectl-wait-ready -- $ns coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run .#wait-for-port-listen -- 1313
    t=$(date +%s)
    nix run .#cli -- set \
        -m ./{{ workspace_dir }}/manifest.json \
        -c localhost:1313 \
        ./{{ workspace_dir }}/deployment/*.yml
    duration=$(( $(date +%s) - $t ))
    echo "Set manifest in $duration seconds."
    echo "set $duration" >> ./{{ workspace_dir }}/just.perf

# Verify the Coordinator.
verify:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{ workspace_dir }}/verify
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run .#kubectl-wait-ready -- $ns coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1314:1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run .#wait-for-port-listen -- 1314
    t=$(date +%s)
    nix run .#cli -- verify \
        -c localhost:1314 \
        -o ./{{ workspace_dir }}/verify
    duration=$(( $(date +%s) - $t ))
    echo "Verified in $duration seconds."
    echo "verify $duration" >> ./{{ workspace_dir }}/just.perf

# Wait for workloads to become ready.
wait-for-workload target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    case {{ target }} in
        "simple")
            nix run .#kubectl-wait-ready -- $ns workload
        ;;
        "openssl")
            nix run .#kubectl-wait-ready -- $ns openssl-backend
            nix run .#kubectl-wait-ready -- $ns openssl-client
            nix run .#kubectl-wait-ready -- $ns openssl-frontend
        ;;
        "emojivoto")
            nix run .#kubectl-wait-ready -- $ns emoji-svc
            nix run .#kubectl-wait-ready -- $ns vote-bot
            nix run .#kubectl-wait-ready -- $ns voting-svc
            nix run .#kubectl-wait-ready -- $ns web-svc
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
        --resource-group "nunki-ci" \
        --name "nunki-ci" \
        --admin

# Destroy a running AKS cluster.
destroy:
    nix run .#destroy-coco-aks -- --name="$azure_resource_group"

# Run code generators.
codegen:
    nix run .#generate

# Format code.
fmt:
    nix fmt

# Lint code.
lint:
    nix run .#golangci-lint -- run

demodir:
    #!/usr/bin/env bash
    d=$(mktemp -d)
    echo "Creating demo directory at ${d}"
    nix build .#nunki.cli
    cp ./result-cli/bin/cli "${d}/nunki"
    cp -R ./deployments/emojivoto "${d}/deployment"
    nix run .#patch-nunki-image-hashes -- "${d}/deployment"
    nix run .#kypatch images -- "${d}/deployment" \
        --replace ghcr.io/edgelesssys ${container_registry}
    echo "Demo directory ready at ${d}"

# Cleanup auxiliary files, caches etc.
clean: undeploy
    rm -rf ./{{ workspace_dir }}
    rm -rf ./layers_cache

# Template for the justfile.env file.

rctemplate := '''
# Container registry to push images to
container_registry=""
# Azure resource group/ resource name. Resource group will be created.
azure_resource_group=""
# Namespace suffix, can be empty. Will be used when patching namespaces.
namespace_suffix=""
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
