# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ pkgs }:

let
  inherit (pkgs.lib) makeScope;
in

makeScope pkgs.pkgsStatic.newScope (
  self:
  pkgs.lib.packagesFromDirectoryRecursive {
    inherit (self) callPackage newScope;
    directory = ./by-name;
  }
)
