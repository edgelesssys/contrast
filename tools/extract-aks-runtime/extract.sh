#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

readonly S3_BUCKET_URI="s3://cdn-confidential-cloud-backend/contrast/node-components"
readonly CDN_URI="https://cdn.confidential.cloud/contrast/node-components"

function cleanup() {
  kubectl delete -f "$podFile" || true
  rm -rf "$tmpdir" || true
}

trap "cleanup" EXIT

podFile="$(dirname "${BASH_SOURCE[0]}")/pod.yml"
kubectl apply -f "$podFile"
kubectl wait --for=condition=Ready pod/extract-aks-runtime --timeout=5m

files=(
  "/opt/confidential-containers/share/kata-containers/kata-containers.img"
  "/opt/confidential-containers/share/kata-containers/kata-containers-igvm.img"
  "/usr/bin/cloud-hypervisor-cvm"
  "/usr/local/bin/containerd-shim-kata-cc-v2"
)

unixtime=$(date +%s)

tmpdir=$(mktemp -d)
mkdir -p "$tmpdir/hashdir"

for file in "${files[@]}"; do
  fileName=$(basename "$file")
  kubectl cp --retries 3 "extract-aks-runtime:/host$file" "$tmpdir/$fileName" >/dev/null 2>&1
  aws s3 cp "$tmpdir/$fileName" "$S3_BUCKET_URI/$unixtime/$fileName" >/dev/null 2>&1
  nixHash=$(nix hash file "$tmpdir/$fileName")
  echo -e "\nurl = \"$CDN_URI/$unixtime/$fileName\";\nhash = \"$nixHash\";\n"
done
