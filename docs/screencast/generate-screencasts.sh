#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

#
# This script prepares the environment for expect scripts to be recorded in,
# executes all scripts, and copies the .cast files to our doc's asset folder.
#

set -euo pipefail

# Setup.

demodir=$(just demodir)
docker build -t screenrecodings docker

# Screencast.
docker run -it \
  -v "${HOME}/.kube/config:/root/.kube/config" \
  -v "$(pwd)/recordings:/recordings" \
  -v "${demodir}:/demo" \
  -v "${demodir}/contrast:/usr/local/bin/contrast" \
  -v "$(pwd)/scripts:/scripts" \
  screenrecodings /scripts/flow.expect

# Cleanup.
kubectl delete -f "${demodir}/deployment/"
kubectl delete -f "${demodir}/coordinator.yaml"
