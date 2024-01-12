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

exitcode=0

az aks nodepool delete \
  --resource-group "${name}" \
  --name nodepool2 \
  --cluster-name "${name}" ||
  exitcode=$?

az aks delete \
  --resource-group "${name}" \
  --name "${name}" \
  --yes ||
  exitcode=$?

exit "${exitcode}"
