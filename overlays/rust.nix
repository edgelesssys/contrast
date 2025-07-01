# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ inputs }:
final: prev: {
  fenix = inputs.fenix.packages.${final.system};
  craneLib = inputs.crane.mkLib prev.pkgs;
}
