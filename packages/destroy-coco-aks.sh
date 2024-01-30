#!/usr/bin/env bash

set -euo pipefail
set -x

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

az aks delete \
  --resource-group "${name}" \
  --name "${name}" \
  --yes
