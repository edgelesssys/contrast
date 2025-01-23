#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

declare -a resources

keepResources=(
  "/subscriptions/0d202bbb-4fa7-4af8-8125-58c269a05435/resourceGroups/contrast-ci/providers/Microsoft.ContainerService/managedClusters/contrast-ci"
  "/subscriptions/0d202bbb-4fa7-4af8-8125-58c269a05435/resourceGroups/contrast-ci/providers/Microsoft.Compute/galleries/contrast_ci_contrast"
  "/subscriptions/0d202bbb-4fa7-4af8-8125-58c269a05435/resourceGroups/contrast-ci/providers/Microsoft.Compute/galleries/contrast_ci_contrast/images/contrast"
)

while IFS= read -r resourceID; do
  for keepResource in "${keepResources[@]}"; do
    if [[ $resourceID == "$keepResource" ]]; then
      continue 2
    fi
  done
  resources+=("$resourceID")
done < <(
  az resource list --resource-group contrast-ci -o json |
    jq -r '
            .[]
            | select(
                .createdTime and (
                    .createdTime
                    | sub("\\.[0-9]+\\+00:00$"; "Z")
                    | fromdateiso8601 < (now - 604800)
                )
            )
            | .id'
)

# Sort resource IDs for better readability
mapfile -t resources < <(printf '%s\n' "${resources[@]}" | sort)

echo Found ${#resources[@]} resources to delete

exitcode=0
for resource in "${resources[@]}"; do
  exitSingle=0
  echo "Deleting resource $resource"
  az resource delete --ids "$resource" || exitSingle=$?
  if [[ $exitSingle -ne 0 ]]; then
    echo "Error: failed to delete resource $resource"
    exitcode=1
  fi
done

exit $exitcode
