# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ contrast }:

contrast.overrideAttrs (prevAttrs: {
  version = prevAttrs.version + "+enterprise";
  tags = prevAttrs.tags ++ [ "enterprise" ];
})
