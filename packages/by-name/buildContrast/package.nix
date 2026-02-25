# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  newScope,
}:

{
}@args:

lib.makeScope newScope (
  self:
  lib.packagesFromDirectoryRecursive {
    inherit (self) callPackage newScope;
    directory = ./../../contrast;
  }
  // args
)
