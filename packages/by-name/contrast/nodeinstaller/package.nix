# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ contrast }:
contrast.nodeinstaller.overrideAttrs (prevAttrs: {
  meta = prevAttrs.meta // {
    mainProgram = "node-installer";
  };
})
