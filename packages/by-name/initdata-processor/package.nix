# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  buildGoModuleSbom,
  runCommand,
  kata,
}:

let
  # Builds a policy that responds to all known agent requests with a fixed response.
  # Such a policy is useful for propagating errors from the guest to the runtime.
  # The actual message needs to be appended to the rules, similar to policy_data in genpolicy.
  deny-with-message = runCommand "deny-with-message" { } ''
    awk '
      # header
      BEGIN { print "package agent_policy\nimport future.keywords.if\n" }

      # default deny rule for the request
      /^message.*Request/ { print "default",$2,":= false" }
      # denying rule that calls print_message
      /^message.*Request/ { print $2,"if { print_message }" }

      # implementation of print_message (`message := "foo"` needs to be appended by the user)
      END { print "\nprint_message if {\n  print(\"Internal error:\", message)\n  false\n}" }
      ' \
      ${kata.runtime.src}/src/libs/protocols/protos/agent.proto >$out
  '';

in

buildGoModule (finalAttrs: {
  pname = "initdata-processor";
  version = builtins.readFile ../../../version.txt;

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "initdata-processor/go.mod")
        (path.append root "initdata-processor/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/attestation"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/atls/validators"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/oid"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/initdata"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "initdata-processor"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-jy/lalOFOp32FGJcAnl+4kHBm3l/U5qichm4WOjAAMI=";

  sourceRoot = "${finalAttrs.src.name}/initdata-processor";
  subPackages = [ "." ];

  prePatch = ''
    install -D ${deny-with-message} policy/assets/deny-with-message.rego
  '';

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  # The policy package embeds deny-with-message.rego, installed in prePatch, so cyclonedx-gomod needs to run after.
  passthru.bombonVendoredSbom = buildGoModuleSbom {
    package = finalAttrs.finalPackage;
    preAnalyze = finalAttrs.prePatch;
  };

  meta = lib.contrast.ourMeta { mainProgram = "initdata-processor"; };
})
