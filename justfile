# Undeploy, rebuild, deploy.
default: undeploy coordinator initializer deploy

# Build the coordinator, containerize and push it.
coordinator:
    nix run .#push-coordinator -- "$container_registry/coordinator-kbs:latest"

# Build the initializer, containerize and push it.
initializer:
    nix run .#push-initializer -- "$container_registry/initializer:latest"

default_deploy_target := "simple"

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target: generate apply

# Generate policies, update manifest.
generate target=default_deploy_target:
    nix run .#yq-go -- -i ". \
        | with(select(.spec.template.spec.containers[].image | contains(\"coordinator-kbs\")); \
        .spec.template.spec.containers[0].image = \"${container_registry}/coordinator-kbs:latest\")" \
        ./deployments/{{target}}/coordinator.yml
    nix run .#yq-go -- -i ". \
        | with(select(.spec.template.spec.initContainers[].image | contains(\"initializer\")); \
        .spec.template.spec.initContainers[0].image = \"${container_registry}/initializer:latest\")" \
        ./deployments/{{target}}/initializer.yml
    nix run .#cli -- generate \
        -m data/manifest.json \
        -p tools \
        -s genpolicy-msft.json \
        ./deployments/{{target}}/{coordinator,initializer}.yml

# Apply Kubernetes manifests from /deployment
apply target=default_deploy_target:
    kubectl apply -f ./deployments/{{target}}/ns.yml
    kubectl apply -f ./deployments/{{target}}/coordinator.yml
    kubectl apply -f ./deployments/{{target}}/initializer.yml
    kubectl apply -f ./deployments/{{target}}/portforwarder.yml

# Delete Kubernetes manifests.
undeploy target=default_deploy_target:
    -kubectl delete -f ./deployments/{{target}}

# Create a CoCo-enabled AKS cluster.
create:
    nix run .#create-coco-aks -- --name="$azure_resource_group"

# Set the manifest at the coordinator.
set:
    #!/usr/bin/env bash
    kubectl -n edg-coco port-forward pod/port-forwarder 1313 &
    PID=$!
    sleep 1
    nix run .#cli -- set -m data/manifest.json -c localhost:1313
    kill $PID


# Destroy a running AKS cluster.
destroy:
    nix run .#destroy-coco-aks -- "$azure_resource_group"

# Cleanup auxiliary files, caches etc.
clean:
    rm -f ./tools/genpolicy.cache/*.{tar,gz,verify}

# Template for the justfile.env file.
rctemplate := '''
# Container registry to push images to
container_registry=""
# Azure resource group/ resource name. Resource group will be created.
azure_resource_group=""
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
