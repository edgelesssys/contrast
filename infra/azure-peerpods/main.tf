terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "4.8.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "3.0.2"
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

data "azurerm_resource_group" "rg" {
  name = local.name
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
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Virtual Machine Contributor"
  principal_id         = azuread_service_principal.sp.object_id
}

resource "azurerm_role_assignment" "ra_reader" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.sp.object_id
}

resource "azurerm_role_assignment" "ra_network_contributor" {
  scope                = data.azurerm_resource_group.rg.id
  role_definition_name = "Network Contributor"
  principal_id         = azuread_service_principal.sp.object_id
}

resource "azuread_application_password" "cred" {
  application_id = azuread_application.app.id
}

resource "azurerm_virtual_network" "main" {
  name                = local.name
  address_space       = ["10.0.0.0/8"]
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name

  subnet {
    name             = "${local.name}_nodenet"
    address_prefixes = ["10.9.0.0/16"]
  }
}

resource "azurerm_kubernetes_cluster" "cluster" {
  name                      = "${local.name}_aks"
  resource_group_name       = data.azurerm_resource_group.rg.name
  node_resource_group       = "${local.name}_node_rg"
  location                  = data.azurerm_resource_group.rg.location
  dns_prefix                = "aks"
  oidc_issuer_enabled       = true
  workload_identity_enabled = true
  sku_tier                  = var.cluster_type

  identity {
    type = "SystemAssigned"
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

data "local_file" "id_rsa" {
  filename = "id_rsa.pub"
}

resource "local_file" "peer-pods-config" {
  filename        = "./peer-pods-config.yaml"
  file_permission = "0777"
  content         = <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: peer-pods-cm
data:
  AZURE_CLIENT_ID: ${azuread_application.app.client_id}
  AZURE_TENANT_ID: ${data.azurerm_subscription.current.tenant_id}
  AZURE_AUTHORITY_HOST: https://login.microsoftonline.com/
  AZURE_IMAGE_ID: ${var.image_id}
  AZURE_INSTANCE_SIZE: Standard_DC2as_v5
  AZURE_REGION: ${data.azurerm_resource_group.rg.location}
  AZURE_RESOURCE_GROUP: ${data.azurerm_resource_group.rg.name}
  AZURE_SUBNET_ID: ${one(azurerm_virtual_network.main.subnet.*.id)}
  AZURE_SUBSCRIPTION_ID: ${data.azurerm_subscription.current.subscription_id}
  CLOUD_PROVIDER: azure
  DISABLECVM: "false"
---
apiVersion: v1
data:
  AZURE_CLIENT_SECRET: ${base64encode(azuread_application_password.cred.value)}
kind: Secret
metadata:
  name: azure-client-secret
---
type: Opaque
apiVersion: v1
data:
  id_rsa.pub: ${data.local_file.id_rsa.content_base64}
kind: Secret
metadata:
  name: ssh-key-secret
type: Opaque
EOF
}
