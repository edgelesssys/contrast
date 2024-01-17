# Undeploy, rebuild, deploy.
default target=default_deploy_target: undeploy coordinator initializer openssl (deploy target) set verify

# Build the coordinator, containerize and push it.
coordinator:
    nix run .#push-coordinator -- "$container_registry/nunki/coordinator:latest"

# Build the openssl container and push it.
openssl:
    nix run .#push-openssl -- "$container_registry/nunki/openssl:latest"

# Build the initializer, containerize and push it.
initializer:
    nix run .#push-initializer -- "$container_registry/nunki/initializer:latest"

default_deploy_target := "simple"
workspace_dir := "workspace"

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target: (generate target) apply

# Generate policies, update manifest.
generate target=default_deploy_target:
    #!/usr/bin/env bash
    mkdir -p ./{{workspace_dir}}
    rm -rf ./{{workspace_dir}}/*
    cp -R ./deployments/{{target}} ./{{workspace_dir}}/deployment
    echo "{{target}}${namespace_suffix}" > ./{{workspace_dir}}/just.namespace
    nix run .#patch-nunki-image-hashes -- ./{{workspace_dir}}/deployment
    nix run .#kypatch images -- ./{{workspace_dir}}/deployment \
        --replace ghcr.io/edgelesssys ${container_registry}
    nix run .#kypatch namespace -- ./{{workspace_dir}}/deployment \
        --replace edg-default {{target}}${namespace_suffix}
    nix run .#cli -- generate \
        -m ./{{workspace_dir}}/manifest.json \
        -p ./{{workspace_dir}} \
        -s genpolicy-msft.json \
        ./{{workspace_dir}}/deployment/*.yml

# Apply Kubernetes manifests from /deployment
apply:
    kubectl apply -f ./{{workspace_dir}}/deployment/ns.yml
    kubectl apply -f ./{{workspace_dir}}/deployment

# Delete Kubernetes manifests.
undeploy:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ ! -d ./{{workspace_dir}} ]]; then
        echo "No workspace directory found, nothing to undeploy."
        exit 0
    fi
    ns=$(cat ./{{workspace_dir}}/just.namespace)
    if kubectl get ns $ns 2> /dev/null; then
        kubectl delete -f ./{{workspace_dir}}/deployment
    fi

# Create a CoCo-enabled AKS cluster.
create:
    nix run .#create-coco-aks -- --name="$azure_resource_group"

# Set the manifest at the coordinator.
set:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{workspace_dir}}/just.namespace)
    nix run .#kubectl-wait-ready -- $ns coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    sleep 1
    nix run .#cli -- set \
        -m ./{{workspace_dir}}/manifest.json \
        -c localhost:1313 \
        ./{{workspace_dir}}/deployment/*.yml

# Verify the Coordinator.
verify:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{workspace_dir}}/verify
    ns=$(cat ./{{workspace_dir}}/just.namespace)
    nix run .#kubectl-wait-ready -- $ns coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    sleep 1
    nix run .#cli -- verify \
        -c localhost:1313 \
        -o ./{{workspace_dir}}/verify

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
    rm -rf ./{{workspace_dir}}
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
