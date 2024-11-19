#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

set -x

if [ -z "${azure_image_id}" ]; then
  nix run -L .#scripts.upload-image -- \
    --subscription-id="${azure_subscription_id:?}" \
    --location="${azure_location:?}" \
    --resource-group="${azure_resource_group:?}"
else
  echo "image_id = \"${azure_image_id}\"" >infra/azure-peerpods/image_id.auto.tfvars
fi

cat >infra/azure-peerpods/e2e.auto.tfvars <<EOF
name_prefix = "${azure_resource_group:?}-$RANDOM-"
EOF

just create AKS-PEER-SNP
just get-credentials AKS-PEER-SNP
just node-installer AKS-PEER-SNP

cleanup() {
  kubectl delete deploy nginx
  kubectl wait --for=delete pod --selector=app=nginx --timeout=5m
}

trap cleanup EXIT

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx
    spec:
      runtimeClassName: kata-remote
      containers:
      - name: nginx
        image: nginx
        imagePullPolicy: Always
EOF

if ! kubectl wait --for=condition=available --timeout=5m deployment/nginx; then
  kubectl describe pods
  exit 1
fi