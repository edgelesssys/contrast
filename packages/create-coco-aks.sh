#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

# Spin up an AKS cluster with CoCo support.
#
# Requires Azure CLI with the aks-preview extension installed.
# Issue for Terraform support:
# https://github.com/hashicorp/terraform-provider-azurerm/issues/24196
#

name=""
location="GermanyWestCentral"
k8sVersion="1.29"

for i in "$@"; do
  case $i in
  --name=*)
    name="${i#*=}"
    shift
    ;;
  --location=*)
    location="${i#*=}"
    shift
    ;;
  --k8s-version=*)
    k8sVersion="${i#*=}"
    shift
    ;;
  *)
    echo "Unknown option $i"
    exit 1
    ;;
  esac
done

set -x

# Will always fail in CI due to lack of permissions.
# In GH actions, CI=true is part of the environment.
az group create \
  --name "${name}" \
  --location "${location}" ||
  $CI

az aks create \
  --resource-group "${name}" \
  --name "${name}" \
  --kubernetes-version "${k8sVersion}" \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --node-count 1 \
  --generate-ssh-keys

az aks nodepool add \
  --resource-group "${name}" \
  --name nodepool2 \
  --cluster-name "${name}" \
  --mode System \
  --node-count 1 \
  --os-sku AzureLinux \
  --node-vm-size Standard_DC4as_cc_v5 \
  --workload-runtime KataCcIsolation

az aks get-credentials \
  --resource-group "${name}" \
  --name "${name}"

az aks nodepool delete \
  --resource-group "${name}" \
  --cluster-name "${name}" \
  --name nodepool1
