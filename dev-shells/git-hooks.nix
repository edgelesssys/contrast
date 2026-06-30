# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  nix-flake-check = {
    enable = true;
    entry = "nix flake check";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
  just-fmt = {
    enable = true;
    entry = "just fmt";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
  just-lint = {
    enable = true;
    entry = "just lint";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
  just-check-links = {
    enable = true;
    entry = "just check-links internal";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
  go-test = {
    enable = true;
    entry = "just unit";
    pass_filenames = false;
    stages = [ "pre-push" ];
  };
}
