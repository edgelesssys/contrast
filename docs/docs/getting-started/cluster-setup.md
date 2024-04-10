# Create a cluster

## Prerequisites

Install the latest version of the [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/).
[Login to your account](https://docs.microsoft.com/en-us/cli/azure/authenticate-azure-cli), which has
the permissions to create an AKS cluster, by executing:

```bash
az login
```

CoCo on AKS is currently in preview. An extension is needed to create such a cluster. Add the
extension with the following commands:

```bash
az extension add \
  --name aks-preview \
  --allow-preview true
az extension update \
  --name aks-preview \
  --allow-preview true
```

Then register the required features:

```bash
az feature register \
    --namespace "Microsoft.ContainerService" \
    --name "KataCcIsolationPreview"
az feature show \
    --namespace "Microsoft.ContainerService" \
    --name "KataCcIsolationPreview"
az provider register \
    --name Microsoft.ContainerService
```

## Set resource group

The AKS with CoCo preview is currently available in the following locations:

```
CentralIndia
eastus
EastUS2EUAP
GermanyWestCentral
japaneast
northeurope
SwitzerlandNorth
UAENorth
westeurope
westus
```

Set the name of the resource group you want to use:

```bash
azResourceGroup="ContrastDemo"
```

You can either use an existing one or create a new resource group with the following command:

```bash
azLocation="westus" # Select a location from the list above

az rg create \
  --name "$azResourceGroup"
  --location "$azLocation"
```

## Create AKS cluster

First create an AKS cluster:

```sh
# Select the name for your AKS cluster
azClusterName="ContrastDemo"

az aks create \
  --resource-group "$azResourceGroup" \
  --name "$azClusterName" \
  --kubernetes-version 1.29 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --node-count 1 \
  --generate-ssh-keys
```

We then add a second node pool with CoCo support:

```bash
az aks nodepool add \
  --resource-group "$azResourceGroup" \
  --name nodepool2 \
  --cluster-name "$azClusterName" \
  --node-count 1 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --workload-runtime KataCcIsolation
```

Finally, update your kubeconfig with the credentials to access the cluster:

```bash
az aks get-credentials \
  --resource-group "$azResourceGroup" \
  --name "$azClusterName"
```

For validation, list the available nodes using kubectl:

```bash
kubectl get nodes
```

It should show two nodes:

```bash
NAME                                STATUS   ROLES    AGE     VERSION
aks-nodepool1-32049705-vmss000000   Ready    <none>   9m47s   v1.29.0
aks-nodepool2-32238657-vmss000000   Ready    <none>   45s     v1.29.0
```
