#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

set -x

if [ -z "${azure_image_id:-}" ]; then
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
just runtime default AKS-PEER-SNP
just apply-runtime default AKS-PEER-SNP

set +x
runtime=$(kubectl get runtimeclass -o json | jq -r '.items[] | .metadata.name | select(startswith("contrast-cc-aks-peer"))')

if [[ $runtime == "" ]]; then
  echo "Contrast RuntimeClass not ready" >&2
  exit 1
fi

kubectl wait "--for=jsonpath={.status.numberReady}=1" ds/contrast-node-installer --timeout=5m

cleanup() {
  kubectl delete deploy alpine
  kubectl wait --for=delete pod --selector=app=alpine --timeout=5m
}

trap cleanup EXIT

run_tests() {
  pod="$(kubectl get pod -l app=alpine -o jsonpath='{.items[0].metadata.name}')"

  # Check IMDS functionality.
  # -f makes this fail on a 500 status code.
  kubectl exec "$pod" -- curl -f -i -H "Metadata: true" http://169.254.169.254/metadata/THIM/amd/certification
}

set -x

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alpine
spec:
  selector:
    matchLabels:
      app: alpine
  replicas: 1
  template:
    metadata:
      labels:
        app: alpine
    spec:
      runtimeClassName: "$runtime"
      containers:
      - name: alpine
        image: alpine/curl
        imagePullPolicy: Always
        command: ["sleep", "infinity"]
EOF

if ! kubectl wait --for=condition=available --timeout=5m deployment/alpine; then
  kubectl describe pods
  kubectl logs -l app.kubernetes.io/name=contrast-node-installer --tail=-1 --all-containers
  exit 1
fi

run_tests
