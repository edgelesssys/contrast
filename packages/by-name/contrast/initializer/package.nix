# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, contrast }:

contrast.initializer.overrideAttrs (_: {
  meta = lib.contrast.ourMeta { mainProgram = "initializer"; };
})
