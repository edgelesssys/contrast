# Undeploy, rebuild, deploy.
default target=default_deploy_target platform=default_platform cli=default_cli: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy (node-installer platform) (deploy target cli platform) set verify (wait-for-workload target)

# Build and push a container image.
push target:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ workspace_dir }}
    pushedImg=$(nix run -L .#containers.push-{{ target }} -- "$container_registry/contrast/{{ target }}")
    printf "ghcr.io/edgelesssys/contrast/%s:latest=%s\n" "{{ target }}" "$pushedImg" >> {{ workspace_dir }}/just.containerlookup

# Build the coordinator, containerize and push it.
coordinator:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ workspace_dir }}

    # We push the coordinator for enterprise/community edition under the same container name for now,
    # as it is convenient regarding the image replacement mechanism.
    # We still have to select the correct nix target, so we cannot use the push target directly.
    if [[ ${enterprise:-} == "true" ]]; then
        nixTarget="containers.push-coordinator-enterprise"
    else
        nixTarget="containers.push-coordinator"
    fi
    pushedImg=$(nix run -L .#${nixTarget} -- "$container_registry/contrast/coordinator")
    printf "ghcr.io/edgelesssys/contrast/%s:latest=%s\n" "coordinator" "$pushedImg" >> {{ workspace_dir }}/just.containerlookup

# Build the openssl container and push it.
openssl: (push "openssl")

# Build the port-forwarder container and push it.
port-forwarder: (push "port-forwarder")

# Build the service-mesh-proxy container and push it.
service-mesh-proxy: (push "service-mesh-proxy")

# Build the initializer, containerize and push it.
initializer: (push "initializer")

# Build the tardev-snapshotter, containerize and push it.
tardev-snapshotter: (push "tardev-snapshotter")

# Build the nydus-snapshotter, containerize and push it.
nydus-snapshotter: (push "nydus-snapshotter")

# Build the nydus-pull container and push it.
nydus-pull: (push "nydus-pull")

default_cli := "contrast.cli"
default_deploy_target := "openssl"
default_platform := "${default_platform}"
workspace_dir := "workspace"

# Build the node-installer, containerize and push it.
node-installer platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            just push "tardev-snapshotter"
            just push "node-installer-microsoft"
        ;;
        "Metal-QEMU-SNP"|"Metal-QEMU-TDX"|"K3s-QEMU-SNP"|"K3s-QEMU-TDX"|"RKE2-QEMU-TDX")
            just push "nydus-snapshotter"
            just push "nydus-pull"
            just push "node-installer-kata"
        ;;
        "Metal-QEMU-SNP-GPU"|"K3s-QEMU-SNP-GPU")
            just push "nydus-snapshotter"
            just push "nydus-pull"
            just push "node-installer-kata-gpu"
        ;;
        "AKS-PEER-SNP")
            nix run -L .#scripts.deploy-caa -- \
                --kustomization=./infra/azure-peerpods/kustomization.yaml \
                --pub-key=./infra/azure-peerpods/id_rsa.pub
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

e2e target=default_deploy_target platform=default_platform: soft-clean coordinator initializer openssl port-forwarder service-mesh-proxy (node-installer platform)
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ "{{ target }}" == "aks-runtime" ]]; then
      echo "WARNING(miampf): The aks-runtime test cannot be executed over just since the confcom azure CLI extension is not installed. Install it first if you want to runt this test over just."
    fi
    nix shell .#contrast.e2e --command {{ target }}.test -test.v \
            --image-replacements ./{{ workspace_dir }}/just.containerlookup \
            --namespace-file ./{{ workspace_dir }}/just.namespace \
            --platform {{ platform }} \
            --namespace-suffix=${namespace_suffix-}

# Generate policies, apply Kubernetes manifests.
deploy target=default_deploy_target cli=default_cli platform=default_platform: (runtime target platform) (apply "runtime") (populate target platform) (generate cli platform) (apply target)

# Populate the workspace with a runtime class deployment
runtime target=default_deploy_target platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p ./{{ workspace_dir }}/runtime
    nix shell .#contrast --command resourcegen \
      --image-replacements ./{{ workspace_dir }}/just.containerlookup \
      --namespace {{ target }}${namespace_suffix-} \
      --add-namespace-object \
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
    fi
    nix shell .#contrast --command resourcegen \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --namespace {{ target }}${namespace_suffix-} \
        --add-port-forwarders \
        --add-logging \
        --add-dmesg \
        --platform {{ platform }} \
        ${target} coordinator > ./{{ workspace_dir }}/deployment/deployment.yml
    echo "{{ target }}${namespace_suffix-}" > ./{{ workspace_dir }}/just.namespace

# Generate policies, update manifest.
generate cli=default_cli platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    nix run -L .#{{ cli }} -- generate \
        --workspace-dir ./{{ workspace_dir }} \
        --image-replacements ./{{ workspace_dir }}/just.containerlookup \
        --reference-values {{ platform }}\
        ./{{ workspace_dir }}/deployment/*.yml

    # On baremetal SNP, we don't have default values for MinimumTCB, so we need to set some here.
    case {{ platform }} in
        "Metal-QEMU-SNP"|"Metal-QEMU-SNP-GPU"|"K3s-QEMU-SNP"|"K3s-QEMU-SNP-GPU")
            yq --inplace \
            '.ReferenceValues.snp.[].MinimumTCB = {"BootloaderVersion":0,"TEEVersion":0,"SNPVersion":0,"MicrocodeVersion":0}' \
            {{ workspace_dir }}/manifest.json
        ;;
        "Metal-QEMU-TDX"|"K3s-QEMU-TDX" | "RKE2-QEMU-TDX")
            yq --inplace \
            '.ReferenceValues.tdx.[].MinimumTeeTcbSvn = "04010200000000000000000000000000" | .ReferenceValues.tdx.[].MrSeam = "1cc6a17ab799e9a693fac7536be61c12ee1e0fabada82d0c999e08ccee2aa86de77b0870f558c570e7ffe55d6d47fa04"' \
            {{ workspace_dir }}/manifest.json
        ;;
    esac

# Apply Kubernetes manifests from /deployment
apply target=default_deploy_target:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ target }} in
        "runtime")
            kubectl apply -f ./{{ workspace_dir }}/runtime
        ;;
        *)
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

upload-image:
    nix run -L .#scripts.upload-image -- --subscription-id="$azure_subscription_id" --location="$azure_location" --resource-group="${azure_resource_group}_caa_cluster"

# Create foundational dependencies of a CoCo-enabled cluster.
create-pre platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            # TODO(burgerdev): this should create the resource group for consistency
            :
        ;;
        "Metal-QEMU-SNP"|"Metal-QEMU-TDX"|"Metal-QEMU-SNP-GPU"|"K3s-QEMU-SNP"|"K3s-QEMU-SNP-GPU"|"K3s-QEMU-TDX"|"RKE2-QEMU-TDX")
            :
        ;;
        "AKS-PEER-SNP")
            echo "resource_group = \"${azure_resource_group}_caa_cluster\"" > infra/azure-peerpods-iam/just.auto.tfvars
            echo "location = \"${azure_location}\"" >> infra/azure-peerpods-iam/just.auto.tfvars
            echo "subscription_id = \"${azure_subscription_id}\"" >> infra/azure-peerpods-iam/just.auto.tfvars
            nix run -L .#terraform -- -chdir=infra/azure-peerpods-iam init
            nix run -L .#terraform -- -chdir=infra/azure-peerpods-iam apply --auto-approve
            nix run -L .#terraform -- -chdir=infra/azure-peerpods-iam output -raw client_secret_env > infra/azure-peerpods/iam.auto.tfvars
            echo "resource_group = \"${azure_resource_group}_caa_cluster\"" >> infra/azure-peerpods/iam.auto.tfvars

            # TODO(burgerdev): this should be done in a generic upload target, together with OCI images
            just upload-image
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Create a CoCo-enabled AKS cluster.
create platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            nix run -L .#scripts.create-coco-aks -- --name="$azure_resource_group" --location="$azure_location"
        ;;
        "Metal-QEMU-SNP"|"Metal-QEMU-TDX"|"Metal-QEMU-SNP-GPU"|"K3s-QEMU-SNP"|"K3s-QEMU-SNP-GPU"|"K3s-QEMU-TDX"|"RKE2-QEMU-TDX")
            :
        ;;
        "AKS-PEER-SNP")
            # Populate Terraform variables.
            echo "subscription_id = \"$azure_subscription_id\"" > infra/azure-peerpods/just.auto.tfvars

            nix run -L .#terraform -- -chdir=infra/azure-peerpods init
            nix run -L .#terraform -- -chdir=infra/azure-peerpods apply --auto-approve
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Set the manifest at the coordinator.
set cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run -L .#scripts.kubectl-wait-ready -- $ns coordinator
    nix run -L .#scripts.kubectl-wait-ready -- $ns port-forwarder-coordinator
    kubectl -n $ns port-forward pod/port-forwarder-coordinator 1313 &
    PID=$!
    trap "kill $PID" EXIT
    nix run -L .#scripts.wait-for-port-listen -- 1313
    nix run -L .#{{ cli }} -- set \
        --workspace-dir ./{{ workspace_dir }} \
        -c localhost:1313 \
        ./{{ workspace_dir }}/deployment/*.yml

# Verify the Coordinator.
verify cli=default_cli:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf ./{{ workspace_dir }}/verify
    ns=$(cat ./{{ workspace_dir }}/just.namespace)
    nix run -L .#scripts.kubectl-wait-ready -- $ns coordinator
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
    nix run -L .#scripts.kubectl-wait-ready -- $ns coordinator
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
        "custom")
            :
        ;;
        *)
            echo "Please register workloads of new targets in wait-for-workload"
            exit 1
        ;;
    esac

# Load the kubeconfig for the given platform.
get-credentials platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            nix run -L .#azure-cli -- aks get-credentials \
                --resource-group "$azure_resource_group" \
                --name "$azure_resource_group"
        ;;
        "AKS-PEER-SNP")
            nix run -L .#scripts.merge-kube-config -- ./infra/azure-peerpods/kube.conf
            ;;
        "K3s-QEMU-TDX")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/m50-ganondorf-kubeconf/versions/latest"
        ;;
        "K3s-QEMU-SNP")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/hetzner-ax162-snp-kubeconfig/versions/latest"
        ;;
        "K3s-QEMU-SNP-GPU")
            nix run -L .#scripts.get-credentials "projects/796962942582/secrets/discovery-kubeconf/versions/latest"
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Load the kubeconfig from the CI AKS cluster.
get-credentials-ci:
    nix run -L .#azure-cli -- aks get-credentials \
        --resource-group "contrast-ci" \
        --name "contrast-ci" \
        --admin

# Destroy a running AKS cluster.
destroy platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            nix run -L .#scripts.destroy-coco-aks -- --name="$azure_resource_group"
        ;;
        "K3s-QEMU-SNP"|"K3s-QEMU-SNP-GPU"|"K3s-QEMU-TDX"|"RKE2-QEMU-TDX")
            :
        ;;
        "AKS-PEER-SNP")
            nix run -L .#terraform -- -chdir=infra/azure-peerpods destroy --auto-approve

            # Clean-up cached image ids.
            rm -f ${CONTRAST_CACHE_DIR}/image-upload/*.image-id

        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Destroy foundational dependencies
destroy-post platform=default_platform:
    #!/usr/bin/env bash
    set -euo pipefail
    case {{ platform }} in
        "AKS-CLH-SNP")
            # TODO(burgerdev): this should destroy the resource group for consistency.
            :
        ;;
        "K3s-QEMU-SNP"|"K3s-QEMU-SNP-GPU"|"K3s-QEMU-TDX"|"RKE2-QEMU-TDX")
            :
        ;;
        "AKS-PEER-SNP")
            nix run -L .#terraform -- -chdir=infra/azure-peerpods-iam destroy --auto-approve

            # We just destroyed the resource group, so these IDs are invalid.
            rm -f ${CONTRAST_CACHE_DIR}/image-upload/*.image-id
        ;;
        *)
            echo "Unsupported platform: {{ platform }}"
            exit 1
        ;;
    esac

# Run code generators.
codegen:
    nix run -L .#scripts.generate

# Format code.
fmt:
    nix fmt

# Lint code.
lint:
    nix run -L .#scripts.golangci-lint -- run

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
# Azure resource group/ resource name. Resource group will be created.
azure_resource_group=""
# Platform to deploy on
default_platform="AKS-CLH-SNP"
# Namespace suffix. Will be used when patching namespaces, chose something that identifies you.
namespace_suffix=""
# Whether to use enterprise or community edition.
enterprise="false"

#
# No need to change anything below this line.
#

# Azure location for the resource group and AKS cluster.
azure_location="westeurope"
# Azure subscription id.
azure_subscription_id="0d202bbb-4fa7-4af8-8125-58c269a05435"
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
