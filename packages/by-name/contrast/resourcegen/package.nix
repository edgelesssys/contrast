# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ contrast }:
(contrast.overrideAttrs (prevAttrs: {
  meta = prevAttrs.meta // {
    mainProgram = "resourcegen";
  };
})).out
