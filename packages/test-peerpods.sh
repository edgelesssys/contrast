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

set +x
found=false
for _ in $(seq 30); do
  if kubectl get runtimeclass | grep -q kata-remote; then
    found=true
    break
  fi
  echo "Waiting for Kata installation to succeed ..."
  sleep 10
done

if [[ $found != true ]]; then
  echo "Kata RuntimeClass not ready" >&2
  exit 1
fi

run_tests() {
  pod="$(kubectl get pod -l app=alpine -o jsonpath='{.items[0].metadata.name}')"

  # Check IMDS functionality.
  # -f makes this fail on a 500 status code.
  kubectl exec "$pod" -- curl -f -i -H "Metadata: true" http://169.254.169.254/metadata/THIM/amd/certification
}

cleanup() {
  kubectl delete deploy alpine
  kubectl wait --for=delete pod --selector=app=alpine --timeout=5m
}

trap cleanup EXIT

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
      runtimeClassName: kata-remote
      containers:
      - name: alpine
        image: alpine/curl
        imagePullPolicy: Always
        command: ["sleep", "3600"]
EOF

if ! kubectl wait --for=condition=available --timeout=5m deployment/alpine; then
  kubectl describe pods
  kubectl logs -n confidential-containers-system -l app=cloud-api-adaptor --tail=-1 --all-containers
  exit 1
fi

run_tests
