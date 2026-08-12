# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ kernel-uvm, ... }@args:

kernel-uvm.override (removeAttrs args [ "kernel-uvm" ] // { withGPU = true; })
