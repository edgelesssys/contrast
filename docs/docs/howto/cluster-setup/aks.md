# Create a cluster

## Prerequisites

- Install version 2.44.1 or newer of the [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/). Note that your package manager will likely install an outdated version.
- Install a recent version of [kubectl](https://kubernetes.io/docs/tasks/tools/).

## Prepare using the AKS preview

First, log in to your Azure subscription:

```bash
az login
```

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

Also enable the feature flag to disable SSH access to the AKS node (recommended, not required):

```bash
az feature register \
  --namespace "Microsoft.ContainerService" \
  --name "DisableSSHPreview"
```

The registration can take a few minutes. The status of the operation can be checked with the following
command, which should show the registration state as `Registered`:

```sh
az feature show \
    --namespace "Microsoft.ContainerService" \
    --name "KataCcIsolationPreview" \
    --output table
az feature show \
    --namespace "Microsoft.ContainerService" \
    --name "DisableSSHPreview" \
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

First, create a CoCo enabled AKS cluster with:

```sh
# Select the name for your AKS cluster
azClusterName="ContrastDemo"

az aks create \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}" \
  --kubernetes-version 1.30 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --workload-runtime KataCcIsolation \
  --node-count 1 \
  --ssh-access disabled
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

It should show a single node:

```bash
NAME                                STATUS   ROLES    AGE     VERSION
aks-nodepool1-32049705-vmss000000   Ready    <none>   9m47s   v1.29.0
```

🥳 Congratulations. You're now ready to set up your first application with Contrast. Follow this [example](../../getting-started/overview.md) to learn how.

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
