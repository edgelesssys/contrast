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

set dotenv-filename := "justfile.env"
set dotenv-load := true
set shell := ["bash", "-uc"]
