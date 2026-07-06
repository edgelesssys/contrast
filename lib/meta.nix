# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib }:

rec {
  org = "github.com/edgelesssys";
  vcs = "https://${org}/contrast";

  supplier = {
    name = "Edgeless Systems GmbH";
    url = [ "https://www.edgeless.systems" ];
    contact = [
      {
        name = "Edgeless Systems GmbH";
        email = "contact@edgeless.systems";
      }
    ];
  };

  ourMeta =
    {
      mainProgram ? null,
      description ? null,
      license ? lib.licenses.bsl11,
    }:
    {
      inherit license vcs;
      maintainers = map (c: { inherit (c) name email; }) supplier.contact;
    }
    // lib.optionalAttrs (mainProgram != null) { inherit mainProgram; }
    // lib.optionalAttrs (description != null) { inherit description; };
}
