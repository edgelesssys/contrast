#!/usr/bin/env bash

export AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
export AZURE_REGION="westeurope"
export AZURE_RESOURCE_GROUP="otto-coco"

az group create --name "${AZURE_RESOURCE_GROUP}" --location "${AZURE_REGION}"


export CLUSTER_NAME="caa-$(date '+%Y%m%b%d%H%M%S')"
# export CLUSTER_NAME="caa-202409Sep09115811"
export AKS_WORKER_USER_NAME="azuser"
export AKS_RG="${AZURE_RESOURCE_GROUP}-aks"
export SSH_KEY=~/.ssh/id_rsa.pub

az aks create \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --node-resource-group "${AKS_RG}" \
  --name "${CLUSTER_NAME}" \
  --enable-oidc-issuer \
  --enable-workload-identity \
  --location "${AZURE_REGION}" \
  --node-count 1 \
  --node-vm-size Standard_B2ms \
  --nodepool-labels node.kubernetes.io/worker= \
  --ssh-key-value "${SSH_KEY}" \
  --admin-username "${AKS_WORKER_USER_NAME}" \
  --os-sku Ubuntu \
  --tier standard

az aks get-credentials \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name "${CLUSTER_NAME}"

export AZURE_WORKLOAD_IDENTITY_NAME="caa-identity"

az identity create \
  --name "${AZURE_WORKLOAD_IDENTITY_NAME}" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --location "${AZURE_REGION}"

export USER_ASSIGNED_CLIENT_ID="$(az identity show \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --name "${AZURE_WORKLOAD_IDENTITY_NAME}" \
  --query 'clientId' \
  -otsv)"

az role assignment create \
  --role "Virtual Machine Contributor" \
  --assignee "$USER_ASSIGNED_CLIENT_ID" \
  --scope "/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourcegroups/${AZURE_RESOURCE_GROUP}"

az role assignment create \
  --role "Reader" \
  --assignee "$USER_ASSIGNED_CLIENT_ID" \
  --scope "/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourcegroups/${AZURE_RESOURCE_GROUP}"

az role assignment create \
  --role "Network Contributor" \
  --assignee "$USER_ASSIGNED_CLIENT_ID" \
  --scope "/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourcegroups/${AKS_RG}"

export AKS_OIDC_ISSUER="$(az aks show \
  --name "$CLUSTER_NAME" \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --query "oidcIssuerProfile.issuerUrl" \
  -otsv)"

az identity federated-credential create \
  --name caa-fedcred \
  --identity-name caa-identity \
  --resource-group "${AZURE_RESOURCE_GROUP}" \
  --issuer "${AKS_OIDC_ISSUER}" \
  --subject system:serviceaccount:confidential-containers-system:cloud-api-adaptor \
  --audience api://AzureADTokenExchange

export AZURE_VNET_NAME=$(az network vnet list \
  --resource-group "${AKS_RG}" \
  --query "[0].name" \
  --output tsv)

export AZURE_SUBNET_ID=$(az network vnet subnet list \
  --resource-group "${AKS_RG}" \
  --vnet-name "${AZURE_VNET_NAME}" \
  --query "[0].id" \
  --output tsv)

export CAA_VERSION="0.9.0"
# curl -LO "https://github.com/confidential-containers/cloud-api-adaptor/archive/refs/tags/v${CAA_VERSION}.tar.gz"
# tar -xvzf "v${CAA_VERSION}.tar.gz"
cd "cloud-api-adaptor-${CAA_VERSION}/src/cloud-api-adaptor"
# export AZURE_IMAGE_ID="/subscriptions/0d202bbb-4fa7-4af8-8125-58c269a05435/resourceGroups/otto-dev/providers/Microsoft.Compute/galleries/cocopriv/images/coco-gpus/versions/0.0.10"
export AZURE_IMAGE_ID="/CommunityGalleries/cococommunity-42d8482d-92cd-415b-b332-7648bd978eff/Images/peerpod-podvm-ubuntu2204-cvm-snp/Versions/0.8.2"
export CAA_IMAGE="quay.io/confidential-containers/cloud-api-adaptor"
export CAA_TAG="v0.9.0-amd64"

cat <<EOF > install/overlays/azure/workload-identity.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cloud-api-adaptor-daemonset
  namespace: confidential-containers-system
spec:
  template:
    metadata:
      labels:
        azure.workload.identity/use: "true"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloud-api-adaptor
  namespace: confidential-containers-system
  annotations:
    azure.workload.identity/client-id: "$USER_ASSIGNED_CLIENT_ID"
EOF

# export AZURE_INSTANCE_SIZE="Standard_NCC40ads_H100_v5"
export AZURE_INSTANCE_SIZE="Standard_DC4as_v5"
export DISABLECVM="false"

cat <<EOF > install/overlays/azure/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
bases:
- ../../yamls
images:
- name: cloud-api-adaptor
  newName: "${CAA_IMAGE}"
  newTag: "${CAA_TAG}"
generatorOptions:
  disableNameSuffixHash: true
configMapGenerator:
- name: peer-pods-cm
  namespace: confidential-containers-system
  literals:
  - CLOUD_PROVIDER="azure"
  - AZURE_SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID}"
  - AZURE_REGION="${AZURE_REGION}"
  - AZURE_INSTANCE_SIZE="${AZURE_INSTANCE_SIZE}"
  - AZURE_RESOURCE_GROUP="${AZURE_RESOURCE_GROUP}"
  - AZURE_SUBNET_ID="${AZURE_SUBNET_ID}"
  - AZURE_IMAGE_ID="${AZURE_IMAGE_ID}"
  - DISABLECVM="${DISABLECVM}"
secretGenerator:
- name: peer-pods-secret
  namespace: confidential-containers-system
- name: ssh-key-secret
  namespace: confidential-containers-system
  files:
  - id_rsa.pub
patchesStrategicMerge:
- workload-identity.yaml
EOF

cp $SSH_KEY install/overlays/azure/id_rsa.pub

export COCO_OPERATOR_VERSION="0.9.0"
kubectl apply -k "github.com/confidential-containers/operator/config/release?ref=v${COCO_OPERATOR_VERSION}"
kubectl apply -k "github.com/confidential-containers/operator/config/samples/ccruntime/peer-pods?ref=v${COCO_OPERATOR_VERSION}"

kubectl apply -k "cloud-api-adaptor-0.9.0/src/cloud-api-adaptor/install/overlays/azure"

# az aks update -g ${AZURE_RESOURCE_GROUP} --name ${CLUSTER_NAME} --disable-file-driver
az aks update -g ${AZURE_RESOURCE_GROUP} --name ${CLUSTER_NAME} --disable-disk-driver

OBJECT_ID="$(az ad sp list --display-name "${CLUSTER_NAME}-agentpool" --query '[].id' --output tsv)"
az role assignment create \
  --role "Contributor" \
  --assignee-object-id ${OBJECT_ID} \
  --assignee-principal-type ServicePrincipal \
  --scope "/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${AZURE_RESOURCE_GROUP}-aks"

exit 0
pushd /home/xcv/repos/azurefile-csi-driver
bash ./deploy/install-driver.sh master local
popd

kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/crd/peerpodvolume.yaml

# azure file
# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/azure-files-csi-wrapper-runner.yaml
# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/azure-files-csi-wrapper-podvm.yaml

# kubectl patch deploy csi-azurefile-controller -n kube-system --patch-file cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/patch-controller.yaml
# kubectl -n kube-system delete replicaset -l app=csi-azurefile-controller
# kubectl patch ds csi-azurefile-node -n kube-system --patch-file cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/patch-node.yaml

# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/azure-file-StorageClass-for-peerpod.yaml
# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/my-pvc.yaml
# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/file/nginx-kata-with-my-pvc-and-csi-wrapper.yaml
# --- azure file end ---

# azure disk
pushd /home/xcv/repos/azuredisk-csi-driver
bash ./deploy/install-driver.sh master local
popd

kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/azure-disk-csi-wrapper-runner.yaml
kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/azure-disk-csi-wrapper-podvm.yaml

kubectl patch deploy csi-azuredisk-controller -n kube-system --patch-file cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/patch-controller.yaml
kubectl -n kube-system delete replicaset -l app=csi-azuredisk-controller
kubectl patch ds csi-azuredisk-node -n kube-system --patch-file cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/patch-node.yaml

kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/azure-disk-storageclass-for-peerpod.yaml
kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/my-pvc.yaml
# kubectl apply -f cloud-api-adaptor-0.9.0/src/csi-wrapper/examples/azure/disk/nginx-kata-with-my-pvc-and-csi-wrapper.yaml
# --- azure disk end ---
