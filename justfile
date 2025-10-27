# Undeploy, rebuild, deploy.
default target=default_deploy_target platform=default_platform cli=default_cli: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump (node-installer platform) (deploy target cli platform) set verify (wait-for-workload target)

# Build and push a container image.
push target:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ workspace_dir }}
    pushedImg=$(nix run -L .#containers.push-{{ target }} -- "$container_registry/contrast/{{ target }}")
    printf "ghcr.io/edgelesssys/contrast/%s:latest=%s\n" "{{ target }}" "$pushedImg" >> {{ workspace_dir }}/just.containerlookup

# Build the coordinator, containerize and push it.
coordinator: (push "coordinator")

# Build the openssl container and push it.
openssl: (push "openssl")

# Build the port-forwarder container and push it.
port-forwarder: (push "port-forwarder")

# Build the service-mesh-proxy container and push it.
service-mesh-proxy: (push "service-mesh-proxy")

# Build the initializer, containerize and push it.
initializer: (push "initializer")

# Build the memdump container and push it.
memdump: (push "memdump")

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

e2e target=default_deploy_target platform=default_platform: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy memdump (node-installer platform)
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ {{ platform }} == "Metal-QEMU-SNP-GPU" ]] ; then
        just request-fifo-ticket 90m
    fi
    nix shell .#contrast.e2e --command {{ target }}.test -test.v \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace-file ./{{ workspace_dir }}/just.namespace \
            --platform {{ platform }} \
            --node-installer-target-conf-type ${node_installer_target_conf_type} \
            --namespace-suffix=${namespace_suffix-} \
            --sync-ticket-file ./{{ workspace_dir }}/just.sync-ticket

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target cli=default_cli platform=default_platform: (runtime target platform) (apply "runtime" platform) (populate target platform) (generate cli platform) (apply target platform)

# Populate the workspace with a runtime class deployment
runtime target=default_deploy_target platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}/runtime
    if [[ "${node_installer_target_conf_type}" != "none" ]]; then
        nix shell .#contrast --command resourcegen \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace {{ target }}${namespace_suffix-} \
            --add-namespace-object \
            --node-installer-target-conf-type ${node_installer_target_conf_type} \
            --platform {{ platform }} \
            node-installer-target-conf > ./{{ workspace_dir }}/runtime/target-conf.yml
    fi
    nix shell .#contrast --command resourcegen \
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
    nix shell .#contrast --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --add-port-forwarders \
        --add-logging \
        --add-dmesg \
        --platform {{ platform }} \
        ${target} coordinator >> ./{{ workspace_dir }}/deployment/deployment.yml
    echo "{{ target }}${namespace_suffix-}" > ./{{ workspace_dir }}/just.namespace

# Generate policies, update manifest.
generate cli=default_cli platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    nix run -L .#{{ cli }} -- generate \
        --workspace-dir ./{{ workspace_dir }} \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --reference-values {{ platform }}\
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
