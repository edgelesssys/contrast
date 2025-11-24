# Undeploy, rebuild, deploy.
default target=default_deploy_target platform=default_platform cli=default_cli: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump debugshell (node-installer platform) (deploy target cli platform) set verify (wait-for-workload target)

# Build and push a container image.
push target:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ workspace_dir }}
    echo "Pushing container $container_registry/contrast/{{ target }}"
    pushedImg=$(nix run -L .#containers.push-{{ target }} -- "$container_registry/contrast/{{ target }}")
    printf "ghcr.io/edgelesssys/contrast/%s:latest=%s\n" "{{ target }}" "$pushedImg" >> {{ workspace_dir }}/just.containerlookup

coordinator: (push "coordinator")

openssl: (push "openssl")

port-forwarder: (push "port-forwarder")

service-mesh-proxy: (push "service-mesh-proxy")

initializer: (push "initializer")

memdump: (push "memdump")

debugshell: (push "debugshell")

default_cli := "contrast.cli"
default_deploy_target := "openssl"
default_platform := "${default_platform}"
workspace_dir := "workspace"

# Build the node-installer, containerize and push it.
node-installer platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "Metal-QEMU-SNP"|"Metal-QEMU-TDX")
            just push "node-installer-kata"
        ;;
        "Metal-QEMU-SNP-GPU")
            just push "node-installer-kata-gpu"
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

e2e target=default_deploy_target platform=default_platform: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump debugshell (node-installer platform)
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ {{ platform }} == "Metal-QEMU-SNP-GPU" ]] ; then
        just request-fifo-ticket 90m
    fi
    if [[ -n "$contrast_ghcr_read" ]]; then
        export CONTRAST_GHCR_READ="$contrast_ghcr_read"
    fi
    nix shell .#contrast.e2e --command {{ target }}.test -test.v \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace-file ./{{ workspace_dir }}/just.namespace \
            --platform {{ platform }} \
            --node-installer-target-conf-type ${node_installer_target_conf_type} \
            --namespace-suffix=${namespace_suffix-} \
            --sync-ticket-file ./{{ workspace_dir }}/just.sync-ticket \
            --insecure-enable-debug-shell-access=${debug:-false}

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target cli=default_cli platform=default_platform: (runtime target platform) (write-namespace target) (apply "runtime" platform) (populate target platform) (generate cli platform) (apply target platform)

# Populate the workspace with a runtime class deployment
runtime target=default_deploy_target platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}/runtime
    if [[ "${node_installer_target_conf_type}" != "none" ]]; then
        nix shell .#contrast.resourcegen --command resourcegen \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace {{ target }}${namespace_suffix-} \
            --add-namespace-object \
            --node-installer-target-conf-type ${node_installer_target_conf_type} \
            --platform {{ platform }} \
            node-installer-target-conf > ./{{ workspace_dir }}/runtime/target-conf.yml
    fi
    nix shell .#contrast.resourcegen --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --add-namespace-object \
        --node-installer-target-conf-type ${node_installer_target_conf_type} \
        --platform {{ platform }} \
        runtime > ./{{ workspace_dir }}/runtime/runtime.yml

# Populate the workspace with a Kubernetes deployment
populate target=default_deploy_target platform=default_platform:
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
    nix shell .#contrast.resourcegen --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --add-port-forwarders \
        --add-logging \
        ${dmesgFlag} \
        --platform {{ platform }} \
        ${target} coordinator >> ./{{ workspace_dir }}/deployment/deployment.yml

# Write the namespace so it can be read by other scripts
write-namespace target=default_deploy_target:
    echo "{{ target }}${namespace_suffix-}" > ./{{ workspace_dir }}/just.namespace

# Generate policies, update manifest.
generate cli=default_cli platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    debugFlag=""
    if [[ "${debug:-}" == "true" ]]; then
        debugFlag="--insecure-enable-debug-shell-access"
    fi
    nix run -L .#{{ cli }} -- generate \
        --workspace-dir ./{{ workspace_dir }} \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --reference-values {{ platform }} \
        ${debugFlag} \
        ./{{ workspace_dir }}/deployment/

    # On baremetal SNP, we don't have default values for MinimumTCB, so we need to set some here.
    case {{ platform }} in
        "Metal-QEMU-SNP"|"Metal-QEMU-SNP-GPU")
            cfg=$(mktemp)
            kubectl -n default get cm bm-tcb-specs -o "jsonpath={.data['tcb-specs\.json']}" > "$cfg"
            export CFG="$cfg"
            yq -i '
            (load(strenv(CFG)).snp) as $b
            | .ReferenceValues.snp = [
                (.ReferenceValues.snp | to_entries)[] as $e
                | ($e.value
                    | .MinimumTCB = (($b[$e.key].MinimumTCB) // .MinimumTCB))
                    | .AllowedChipIDs = (($b[$e.key].AllowedChipIDs) // .AllowedChipIDs)
                ]
            ' {{ workspace_dir }}/manifest.json
        ;;
        "Metal-QEMU-TDX")
            cm=$(kubectl get -n default cm bm-tcb-specs -o "jsonpath={.data['tcb-specs\.json']}")
            mrSeam=$(echo "$cm" | yq '.tdx.[].MrSeam') \
                yq -i \
                '.ReferenceValues.tdx.[].MrSeam = strenv(mrSeam)' \
                {{ workspace_dir }}/manifest.json
        ;;
    esac

# Apply Kubernetes manifests from /deployment
apply target=default_deploy_target platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ target }} in
        "runtime")
            if [[ -f ./{{ workspace_dir }}/runtime/target-conf.yml ]]; then
                kubectl apply -f ./{{ workspace_dir }}/runtime/target-conf.yml
            fi
            if [[ -n "$contrast_ghcr_read" ]]; then
                cat > "./{{ workspace_dir }}/contrast-imagepuller.toml" <<EOF
    [registries]
    [registries."ghcr.io."]
    auth = "$(printf "user-not-required-here:%s" "$contrast_ghcr_read" | base64 -w0)"
    EOF
                kubectl create secret generic contrast-node-installer-imagepuller-config \
                    --from-file "contrast-imagepuller.toml"="./{{ workspace_dir }}/contrast-imagepuller.toml" \
                    --namespace $(cat ./{{ workspace_dir }}/just.namespace)
            fi
            kubectl apply -f ./{{ workspace_dir }}/runtime/runtime.yml
        ;;
        *)
            if [[ {{ platform }} == "Metal-QEMU-SNP-GPU" ]] ; then
                just request-fifo-ticket 90m
                trap 'just release-fifo-ticket' ERR
                kubectl label ns $(cat ./{{ workspace_dir }}/just.namespace) contrast.edgeless.systems/sync-ticket=$(cat ./{{ workspace_dir }}/just.sync-ticket) --overwrite
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
    if ! kubectl get ns $ns 2> /dev/null; then
        echo "Namespace $ns does not exist, nothing to undeploy."
        exit 0
    fi
    if [[ -f ./{{ workspace_dir }}/deployment/ns.yml ]]; then
        kubectl delete \
            -f ./{{ workspace_dir }}/deployment \
            --ignore-not-found \
            --grace-period=30 \
            --timeout=10m
    else
        kubectl delete namespace $ns
    fi
    just release-fifo-ticket

# Set the manifest at the coordinator.
set cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run -L .#scripts.kubectl-wait-coordinator -- $ns
    nix run -L .#scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#scripts.wait-for-port-listen -- 1313
    nix run -L .#{{ cli }} -- set \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313 \
        ./{{ workspace_dir }}/deployment/

# Verify the Coordinator.
verify cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{ workspace_dir }}/verify
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run -L .#scripts.kubectl-wait-coordinator -- $ns
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1314:1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#scripts.wait-for-port-listen -- 1314
    nix run -L .#{{ cli }} -- verify \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1314

# Recover the Coordinator.
recover cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run -L .#scripts.kubectl-wait-coordinator -- $ns
    nix run -L .#scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#scripts.wait-for-port-listen -- 1313
    nix run -L .#{{ cli }} -- recover \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313

# Wait for workloads to become ready.
wait-for-workload target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    case {{ target }} in
        "openssl")
            nix run -L .#scripts.kubectl-wait-ready -- $ns openssl-backend
            nix run -L .#scripts.kubectl-wait-ready -- $ns openssl-frontend
        ;;
        "emojivoto" | "emojivoto-sm-ingress")
            nix run -L .#scripts.kubectl-wait-ready -- $ns emoji-svc
            nix run -L .#scripts.kubectl-wait-ready -- $ns vote-bot
            nix run -L .#scripts.kubectl-wait-ready -- $ns voting-svc
            nix run -L .#scripts.kubectl-wait-ready -- $ns web-svc
        ;;
        "volume-stateful-set")
            nix run .#scripts.kubectl-wait-ready -- $ns volume-tester
        ;;
        "mysql")
            nix run .#scripts.kubectl-wait-ready -- $ns mysql-backend
            nix run .#scripts.kubectl-wait-ready -- $ns mysql-client
        ;;
        "vault")
            nix run .#scripts.kubectl-wait-ready -- $ns vault
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
    ticket=$(nix run .#scripts.get-sync-ticket {{ timeout }})
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
    nix run .#scripts.release-sync-ticket ${ticket}
    rm ./{{ workspace_dir }}/just.sync-ticket

# Load the kubeconfig for the given platform.
get-credentials platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "Metal-QEMU-TDX"|"olimar")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/olimar-kubeconfig/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-TDX"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="k3s"/' justfile.env
        ;;
        "Metal-QEMU-SNP"|"palutena")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/palutena-kubeconfig/versions/latest"
            sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-SNP"/' justfile.env
            sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="none"/' justfile.env
        ;;
        "Metal-QEMU-SNP-GPU"|"discovery")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/discovery-kubeconf/versions/latest"
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
    nix run -L .#scripts.get-credentials "projects/796962942582/secrets/hetzner-ax162-snp-kubeconfig/versions/latest"
    sed -i 's/^default_platform=.*/default_platform="Metal-QEMU-SNP"/' justfile.env
    sed -i 's/^node_installer_target_conf_type=.*/node_installer_target_conf_type="k3s"/' justfile.env

# Get the Github token with read access to Contrast's ghcr.io packages.
get-ghcr-read-token:
    #!/usr/bin/env bash
    set -euo pipefail
    token=$(nix run -L .#scripts.get-ghcr-read-token "projects/796962942582/secrets/ghcr-read-token/versions/latest")
    sed -i "s/^contrast_ghcr_read=.*/contrast_ghcr_read=\"${token}\"/" justfile.env

# Run code generators.
codegen:
    nix run -L .#scripts.generate

# Format code.
fmt:
    nix fmt

# Lint code.
lint:
    nix run -L .#scripts.golangci-lint -- run

# Check links.
check-links:
    nix run .#nixpkgs.lychee -- --config tools/lychee/config.toml .

demodir version="latest": undeploy
    #!/usr/bin/env bash
    set -euo pipefail
    v="$(echo {{ version }} | sed 's/\./-/g')"
    nix develop -u DIRENV_DIR -u DIRENV_FILE -u DIRENV_DIFF -u DIRENV_WATCHES .#demo-$v

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
# Platform to deploy on
default_platform="Metal-QEMU-SNP"
# Node installer target config map to deploy.
node_installer_target_conf_type="k3s"
# Enable insecure debug features like debug shell access.
debug="false"

#
# No need to change anything below this line.
#

# Namespace suffix, can be empty. Will be used when patching namespaces.
namespace_suffix=""
# Cache directory for the CLI.
CONTRAST_CACHE_DIR="./workspace.cache"
# Log level for the CLI.
CONTRAST_LOG_LEVEL=""
# A Github token with read access to the Contrast ghcr.io packages.
# Should be set by running just get-ghcr-read-token.
contrast_ghcr_read=""
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
