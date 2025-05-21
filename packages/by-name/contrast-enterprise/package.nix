# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ contrast }:

contrast.overrideAttrs (prevAttrs: {
  version = prevAttrs.version + "+enterprise";
  __intentionallyOverridingVersion = true;
  tags = prevAttrs.tags ++ [ "enterprise" ];
  postInstall =
    prevAttrs.postInstall
    + ''
      mv $cli/bin/contrast $cli/bin/contrast-enterprise
    '';
  meta = prevAttrs.meta // {
    mainProgram = "contrast-enterprise";
  };
})
