# Create a cluster

## Prerequisites

Install the latest version of the [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/).

[Login to your account](https://docs.microsoft.com/en-us/cli/azure/authenticate-azure-cli), which needs
to have the permissions to create an AKS cluster, by executing:

```bash
az login
```

## Prepare using the AKS preview

CoCo on AKS is currently in preview. An extension for the `az` CLI is needed to create such a cluster.
Add the extension with the following commands:

```bash
az extension add \
  --name aks-preview \
  --allow-preview true
az extension update \
  --name aks-preview \
  --allow-preview true
```

Then register the required feature flags in your subscription to allow access to the public preview:

```bash
az feature register \
    --namespace "Microsoft.ContainerService" \
    --name "KataCcIsolationPreview"
```

The registration can take a few minutes. The status of the operation can be checked with the following
command, which should show the registration state as `Registered`:

```sh
az feature show \
    --namespace "Microsoft.ContainerService" \
    --name "KataCcIsolationPreview" \
    --output table
```

Afterward, refresh the registration of the ContainerService provider:

```sh
az provider register \
    --namespace "Microsoft.ContainerService"
```

## Create resource group

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

az group create \
  --name "${azResourceGroup:?}" \
  --location "${azLocation:?}"
```

## Create AKS cluster

First, we need to create an AKS cluster. We can't directly create a CoCo-enabled cluster, so we'll need to create a
non-CoCo cluster first, and then add a CoCo node pool, optionally replacing the non-CoCo node pool.

We'll first start by creating the non-CoCo cluster:

```sh
# Select the name for your AKS cluster
azClusterName="ContrastDemo"

az aks create \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}" \
  --kubernetes-version 1.29 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --node-count 1 \
  --generate-ssh-keys
```

We then add a second node pool with CoCo support:

```bash
az aks nodepool add \
  --resource-group "${azResourceGroup:?}" \
  --name nodepool2 \
  --cluster-name "${azClusterName:?}" \
  --node-count 1 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --workload-runtime KataCcIsolation
```

Optionally, we can now remove the non-CoCo node pool:

```bash
az aks nodepool delete \
  --resource-group "${azResourceGroup:?}" \
  --cluster-name "${azClusterName:?}" \
  --name nodepool1
```

Finally, update your kubeconfig with the credentials to access the cluster:

```bash
az aks get-credentials \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}"
```

For validation, list the available nodes using `kubectl`:

```bash
kubectl get nodes
```

It should show two nodes:

```bash
NAME                                STATUS   ROLES    AGE     VERSION
aks-nodepool1-32049705-vmss000000   Ready    <none>   9m47s   v1.29.0
aks-nodepool2-32238657-vmss000000   Ready    <none>   45s     v1.29.0
```

🥳 Congratulations. You're now ready to set up your first application with Contrast. Follow this [example](../examples/emojivoto.md) to learn how.

## Cleanup

After trying out Contrast, you might want to clean up the cloud resources created in this step.
In case you've created a new resource group, you can just delete that group with

```sh
az group delete \
  --name "${azResourceGroup:?}"
```

Deleting the resource group will also delete the cluster and all other related resources.

To only cleanup the AKS cluster and node pools, run

```sh
az aks delete \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}"
```
