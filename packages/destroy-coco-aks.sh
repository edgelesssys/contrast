#!/usr/bin/env bash

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

az aks delete \
  --resource-group "${name}" \
  --name "${name}" \
  --yes
