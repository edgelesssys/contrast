# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
  kata,
}:

buildGoModule (_finalAttrs: {
  pname = "policy-test";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (fileset.difference (path.append root "policy-test") (path.append root "policy-test/testdata"))
        (path.append root "cli")
        (path.append root "internal")
        (path.append root "sdk")
        (path.append root "apitypes")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-Cob7OieQcJBSc4L4j9X2vi0AYaPU4JdB9Jt8QhIJSk4=";

  modRoot = "policy-test";

  preConfigure = ''
    install -D ${kata.genpolicy.settings-dev}/genpolicy-settings.json policy-test/assets/genpolicy-settings-kata.json
    # Move postConfigure here as well, because the configurePhase already cd's into modRoot.
    ${contrast.cli.postConfigure}
  '';

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
  ];

  tags = [ "contrast_unstable_api" ];

  meta = lib.contrast.ourMeta { mainProgram = "policy-test"; };
})
