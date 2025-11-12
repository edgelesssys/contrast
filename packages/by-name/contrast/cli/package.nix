# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ contrast }:
contrast.cli.overrideAttrs (prevAttrs: {
  meta = prevAttrs.meta // {
    mainProgram = "contrast";
  };
})
