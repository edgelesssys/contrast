#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Usage:
#   configure-imagepuller-mirror contrast-imagepuller.toml

set -euo pipefail

tomlTemplate=$(readlink -f "$1")

workdir=$(mktemp -d)

cleanup() {
  rm -rf "$workdir" || true
  kubectl delete csr contrast-imagepuller-auth || true
}

trap cleanup EXIT

pushd "$workdir"

# Create a client cert that's accepted by the k8s API server, including the mirror endpoint.

openssl ecparam -genkey -name prime256v1 -noout -out imagepuller.key
openssl req -new -key imagepuller.key -subj "/CN=contrast-imagepuller-auth" -out imagepuller.csr

kubectl apply -f - <<EOF
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: contrast-imagepuller-auth
spec:
  request: $(base64 -w0 <imagepuller.csr)
  signerName: kubernetes.io/kube-apiserver-client
  expirationSeconds: 31536000   # 1 year, larger values will be capped by k3s.
  usages: ["client auth"]
EOF

kubectl certificate approve contrast-imagepuller-auth
kubectl wait --for=jsonpath='{.status.certificate}' --timeout=60s csr/contrast-imagepuller-auth
kubectl get csr contrast-imagepuller-auth -o jsonpath='{.status.certificate}' | base64 -d >imagepuller.crt

kubectl -n kube-system get configmap kube-root-ca.crt -o jsonpath='{.data.ca\.crt}' >ca.crt

yq -p toml -o toml '
  .registries["."]["client-cert"] = load_str("imagepuller.crt") |
  .registries["."]["client-key"]  = load_str("imagepuller.key") |
  .registries["."]["ca-certs"]    = load_str("ca.crt")
' "$tomlTemplate" >contrast-imagepuller.toml

kubectl create secret generic contrast-e2e-registry-auth --dry-run=client -o yaml -n default --from-file=contrast-imagepuller.toml=contrast-imagepuller.toml |
  kubectl apply -f -
