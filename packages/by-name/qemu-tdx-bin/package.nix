# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ fetchzip }:

# TODO(msanft): Incorporate the Canonical TDX QEMU patches in our QEMU build for a dynamically
# built SEV / TDX QEMU binary. For now, take the blob from a build of the following, which matches
# what Canonical provides in Ubuntu 24.04.
# https://code.launchpad.net/~kobuk-team/+recipe/tdx-qemu-noble
fetchzip {
  url = "https://cdn.confidential.cloud/contrast/node-components/1%3A8.2.2%2Bds-0ubuntu2%2Btdx1.0~tdx1.202407031834~ubuntu24.04.1/1%3A8.2.2%2Bds-0ubuntu2%2Btdx1.0~tdx1.202407031834~ubuntu24.04.1.zip";
  hash = "sha256-6TztmmmO2N1jk/cNKdvd/MMIf43N7lxPaasjKARRVik=";
}
