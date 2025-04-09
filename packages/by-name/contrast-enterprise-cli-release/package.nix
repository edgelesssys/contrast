# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  contrast-cli-release,
  contrast-enterprise,
}:

contrast-cli-release.override {
  contrast = contrast-enterprise;
}
