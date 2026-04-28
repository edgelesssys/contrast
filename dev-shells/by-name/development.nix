# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  # keep-sorted start
  crane,
  delve,
  git-hooks-lib,
  go,
  golangci-lint,
  gopls,
  gotools,
  just,
  kubectl,
  yq-go,
  # keep-sorted end
  mkShell,
}:

let
  hooks = git-hooks-lib.run {
    src = ../..;
    hooks = import ../git-hooks.nix;
  };
in

mkShell {
  packages = [
    # keep-sorted start
    crane
    delve
    go
    golangci-lint
    gopls
    gotools
    just
    kubectl
    yq-go
    # keep-sorted end
  ]
  ++ hooks.enabledPackages;
  shellHook = ''
    alias make=just
    export DO_NOT_TRACK=1
    export CGO_ENABLED=0
    ${hooks.shellHook}
  '';
}
