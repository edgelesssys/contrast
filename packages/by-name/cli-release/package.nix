{ lib
, contrast
, genpolicy-msft
, genpolicy ? genpolicy-msft
}:

(contrast.overrideAttrs (_finalAttrs: previousAttrs: {
  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/cmd/assets/genpolicy
    install -D ${genpolicy.settings}/genpolicy-settings.json cli/cmd/assets/genpolicy-settings.json
    install -D ${genpolicy.rules}/genpolicy-rules.rego cli/cmd/assets/genpolicy-rules.rego
  '';

  ldflags = previousAttrs.ldflags ++ [
    "-X github.com/edgelesssys/contrast/cli/cmd.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/cmd/assets/coordinator-policy-hash}"
  ];
})).cli
