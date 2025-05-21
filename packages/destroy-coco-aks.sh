#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

for i in "$@"; do
  case $i in
  --name=*)
    name="${i#*=}"
    shift
    ;;
  *)
    echo "Unknown option $i"
    exit 1
    ;;
  esac
done

set -x

if [[ -z ${CI:-} ]]; then
  az group delete \
    --name "${name}" \
    --yes
else
  az aks delete \
    --resource-group "${name}" \
    --name "${name}" \
    --yes
fi
