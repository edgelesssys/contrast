#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

#
# This script prepares the environment for expect scripts to be recorded in,
# executes all scripts, and copies the .cast files to our doc's asset folder.
#

set -euo pipefail

scriptdir=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")
demodir=$(nix develop .#demo-latest --command pwd)
contrastPath=$(nix build .#contrast-releases.latest && realpath result/bin/contrast)
version=$(jq -r '.contrast | last | .version' "../../packages/contrast-releases.json")
sed -i "s#download/.*/#download/$version/#g" ./scripts/flow.expect

docker build -t screenrecodings "${scriptdir}/docker"

docker run -t \
  -v "${HOME}/.kube/config:/root/.kube/config" \
  -v "${scriptdir}/recordings:/recordings" \
  -v "${scriptdir}/scripts:/scripts" \
  -v "${demodir}:/demo" \
  -v "${contrastPath}:/usr/local/bin/contrast" \
  screenrecodings /scripts/flow.expect

kubectl delete -f "${demodir}/deployment/"
kubectl delete -f "${demodir}/coordinator.yml"
kubectl delete -f "${demodir}/runtime.yml"
rm result
