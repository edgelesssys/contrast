# Undeploy, rebuild, deploy.
default: undeploy coordinator initializer deploy

# Build the coordinator, containerize and push it.
coordinator:
    nix run .#push-coordinator -- "$container_registry/coordinator-kbs:latest"

# Build the initializer, containerize and push it.
initializer:
    nix run .#push-initializer -- "$container_registry/initializer:latest"

# Generate policies, apply Kubernetes manifests.
deploy: generate apply

# Generate policies, update manifest.
generate:
    nix run .#yq-go -- -i ". \
        | with(select(.spec.template.spec.containers[].image | contains(\"coordinator-kbs\")); \
        .spec.template.spec.containers[0].image = \"${container_registry}/coordinator-kbs:latest\")" \
        ./deployment/coordinator.yml
    nix run .#yq-go -- -i ". \
        | with(select(.spec.template.spec.initContainers[].image | contains(\"initializer\")); \
        .spec.template.spec.initContainers[0].image = \"${container_registry}/initializer:latest\")" \
        ./deployment/initializer.yml
    nix run .#cli -- generate \
        -m data/manifest.json \
        -p tools \
        -s genpolicy-msft.json \
        ./deployment/{coordinator,initializer}.yml

# Apply Kubernetes manifests from /deployment.
apply:
    kubectl apply -f ./deployment/ns.yml
    kubectl apply -f ./deployment/coordinator.yml
    kubectl apply -f ./deployment/initializer.yml
    kubectl apply -f ./deployment/portforwarder.yml

# Delete Kubernetes manifests.
undeploy:
    -kubectl delete -f ./deployment

# Create a CoCo-enabled AKS cluster.
create:
    nix run .#create-coco-aks -- --name="$azure_resource_group"

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
