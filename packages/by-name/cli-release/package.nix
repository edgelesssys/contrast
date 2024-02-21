{ lib
, nunki
, genpolicy-msft
, genpolicy ? genpolicy-msft
}:

(nunki.overrideAttrs (_finalAttrs: previousAttrs: {
  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/assets/genpolicy
    install -D ${genpolicy.settings}/genpolicy-settings.json cli/assets/genpolicy-settings.json
    install -D ${genpolicy.rules}/genpolicy-rules.rego cli/assets/genpolicy-rules.rego
  '';

  ldflags = previousAttrs.ldflags ++ [
    "-X main.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/assets/coordinator-policy-hash}"
  ];
})).cli
