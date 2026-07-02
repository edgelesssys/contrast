# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib }:

{
  mainProgram ? null,
  description ? null,
  license ? lib.licenses.bsl11,
  homepage ? "https://github.com/edgelesssys/contrast",
  maintainerName ? "Edgeless Systems",
  maintainerEmail ? "contact@edgeless.systems",
}:

{
  inherit license homepage;
  maintainers = [
    {
      name = maintainerName;
      email = maintainerEmail;
    }
  ];
}
// lib.optionalAttrs (mainProgram != null) { inherit mainProgram; }
// lib.optionalAttrs (description != null) { inherit description; }
