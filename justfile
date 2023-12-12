default: undeploy coordinator initializer deploy

coordinator:
    nix run .#push-coordinator -- "$container_registry/coordinator-kbs:latest"

initializer:
    nix run .#push-initializer -- "$container_registry/initializer:latest"

deploy:
    nix run .#cli -- generate \
        -m data/manifest.json \
        -p tools \
        -s genpolicy-msft.json \
        ./deployment/{coordinator,initializer}.yml
    kubectl apply -f ./deployment/ns.yml
    kubectl apply -f ./deployment/coordinator.yml
    kubectl apply -f ./deployment/initializer.yml
    kubectl apply -f ./deployment/portforwarder.yml

undeploy:
    -kubectl delete -f ./deployment

clean:
    rm -f ./tools/genpolicy.cache/*.{tar,gz,verify}


rctemplate := '''
# Container registry to push images to
container_registry=""
'''

onboard:
    @ [[ -f "./justfile.env" ]] && echo "justfile.env already exists" && exit 1 || true
    @echo '{{ rctemplate }}' > ./justfile.env
    @echo "Created ./justfile.env. Please fill it out."

set dotenv-filename := "justfile.env"
set dotenv-load := true
set shell := ["bash", "-uc"]
