terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "4.26.0"
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

locals {
  name = "${var.name_prefix}contrast_pp"
}

data "azurerm_resource_group" "rg" {
  name = var.resource_group
}

resource "azurerm_virtual_network" "main" {
  name                = "${local.name}_net"
  address_space       = ["10.0.0.0/8"]
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name

  subnet {
    name             = "${local.name}_nodenet"
    address_prefixes = ["10.9.0.0/16"]
  }
}

resource "azurerm_public_ip" "nat_ip" {
  name                = "${local.name}_nat_ip"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_nat_gateway" "nat" {
  name                = "${local.name}_nat"
  location            = data.azurerm_resource_group.rg.location
  resource_group_name = data.azurerm_resource_group.rg.name
}

resource "azurerm_nat_gateway_public_ip_association" "ip_to_nat" {
  nat_gateway_id       = azurerm_nat_gateway.nat.id
  public_ip_address_id = azurerm_public_ip.nat_ip.id
}

resource "azurerm_subnet_nat_gateway_association" "subnet_to_nat" {
  subnet_id      = one(azurerm_virtual_network.main.subnet.*.id)
  nat_gateway_id = azurerm_nat_gateway.nat.id
}

resource "azurerm_kubernetes_cluster" "cluster" {
  name                      = "${local.name}_aks"
  resource_group_name       = data.azurerm_resource_group.rg.name
  node_resource_group       = "${local.name}_aks_node_rg"
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
  newTag: v0.10.0-amd64
generatorOptions:
  disableNameSuffixHash: true
configMapGenerator:
- name: peer-pods-cm
  namespace: confidential-containers-system
  literals:
  - CLOUD_PROVIDER=azure
  - AZURE_SUBSCRIPTION_ID=${data.azurerm_subscription.current.subscription_id}
  - AZURE_REGION=${data.azurerm_resource_group.rg.location}
  - AZURE_INSTANCE_SIZE=Standard_DC2as_v5
  - AZURE_RESOURCE_GROUP=${data.azurerm_resource_group.rg.name}
  - AZURE_SUBNET_ID=${one(azurerm_virtual_network.main.subnet.*.id)}
  - AZURE_IMAGE_ID=${var.image_id}
  - AZURE_CLIENT_ID=${var.client_id}
  - AZURE_TENANT_ID=${var.tenant_id}
  - AZURE_CLIENT_SECRET=${var.client_secret}
  - DISABLECVM=false
secretGenerator:
- name: peer-pods-secret
  namespace: confidential-containers-system
- name: ssh-key-secret
  namespace: confidential-containers-system
  files:
  - id_rsa.pub
EOF
}
