{ nunki }:

(nunki.overrideAttrs (_finalAttrs: previousAttrs: {
  ldflags = previousAttrs.ldflags ++ [
    "-X main.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/assets/coordinator-policy-hash}"
  ];
})).cli
