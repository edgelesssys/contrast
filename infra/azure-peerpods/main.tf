terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "4.3.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "2.53.1"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.5.2"
    }
  }
}

provider "azurerm" {
  subscription_id = var.subscription_id
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

data "azurerm_subscription" "current" {}

data "azuread_client_config" "current" {}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

locals {
  name = "${var.name_prefix}_caa_cluster"
}

data "azurerm_resource_group" "rg_podvm_image" {
  name = var.image_resource_group_name
}

resource "azurerm_resource_group" "rg" {
  name     = local.name
  location = "germanywestcentral"
}

resource "azuread_application" "app" {
  display_name = local.name
  owners       = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal" "sp" {
  client_id                    = azuread_application.app.client_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azurerm_role_assignment" "ra_vm_contributor" {
  scope                = azurerm_resource_group.rg.id
  role_definition_name = "Virtual Machine Contributor"
  principal_id         = azuread_service_principal.sp.id
}

resource "azurerm_role_assignment" "ra_reader" {
  scope                = azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.sp.id
}

resource "azurerm_role_assignment" "ra_network_contributor" {
  scope                = azurerm_resource_group.rg.id
  role_definition_name = "Network Contributor"
  principal_id         = azuread_service_principal.sp.id
}

resource "azuread_application_federated_identity_credential" "federated_credentials" {
  display_name   = local.name
  application_id = azuread_application.app.id
  issuer         = azurerm_kubernetes_cluster.cluster.oidc_issuer_url
  subject        = "system:serviceaccount:confidential-containers-system:cloud-api-adaptor"
  audiences      = ["api://AzureADTokenExchange"]
}

resource "azurerm_role_assignment" "ra_image" {
  scope                = data.azurerm_resource_group.rg_podvm_image.id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.sp.id
}

resource "azuread_application_password" "cred" {
  application_id = azuread_application.app.id
}

resource "azurerm_virtual_network" "main" {
  name                = local.name
  address_space       = ["10.0.0.0/8"]
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  subnet {
    name             = "${local.name}_nodenet"
    address_prefixes = ["10.9.0.0/16"]
  }
}

resource "azurerm_kubernetes_cluster" "cluster" {
  name                      = "${local.name}_aks"
  resource_group_name       = azurerm_resource_group.rg.name
  node_resource_group       = "${local.name}_node_rg"
  location                  = azurerm_resource_group.rg.location
  dns_prefix                = "aks"
  oidc_issuer_enabled       = true
  workload_identity_enabled = true

  identity {
    type = "SystemAssigned"
  }

  linux_profile {
    admin_username = "azuser"
    ssh_key { key_data = file(var.ssh_pub_key_path) }
  }

  default_node_pool {
    name                 = "default"
    node_count           = 1
    vm_size              = "Standard_F4s_v2"
    os_sku               = "Ubuntu"
    auto_scaling_enabled = false
    type                 = "VirtualMachineScaleSets"
    vnet_subnet_id       = one(azurerm_virtual_network.main.subnet.*.id)
    node_labels = {
      "node.kubernetes.io/worker"      = ""
      "katacontainers.io/kata-runtime" = "true"
    }
  }
}

resource "local_file" "kubeconfig" {
  depends_on = [azurerm_kubernetes_cluster.cluster]
  filename   = "./kube.conf"
  content    = azurerm_kubernetes_cluster.cluster.kube_config_raw
}

resource "local_file" "workload_identity" {
  filename        = "./workload-identity.yaml"
  file_permission = "0777"
  content         = <<EOF
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
    azure.workload.identity/client-id: ${azuread_application.app.client_id}
EOF
}

resource "local_file" "kustomization" {
  filename        = "./kustomization.yaml"
  file_permission = "0777"
  content         = <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
bases:
- ../../yamls
images:
- name: cloud-api-adaptor
  newName: quay.io/confidential-containers/cloud-api-adaptor
  newTag: v0.9.0-amd64
generatorOptions:
  disableNameSuffixHash: true
configMapGenerator:
- name: peer-pods-cm
  namespace: confidential-containers-system
  literals:
  - CLOUD_PROVIDER=azure
  - AZURE_SUBSCRIPTION_ID=${data.azurerm_subscription.current.subscription_id}
  - AZURE_REGION=${azurerm_resource_group.rg.location}
  - AZURE_INSTANCE_SIZE=Standard_DC2as_v5
  - AZURE_RESOURCE_GROUP=${azurerm_resource_group.rg.name}
  - AZURE_SUBNET_ID=${one(azurerm_virtual_network.main.subnet.*.id)}
  - AZURE_IMAGE_ID=/subscriptions/0d202bbb-4fa7-4af8-8125-58c269a05435/resourceGroups/otto-dev/providers/Microsoft.Compute/galleries/cocopriv/images/coco-gpus/versions/0.0.33
  - DISABLECVM=false
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
}
