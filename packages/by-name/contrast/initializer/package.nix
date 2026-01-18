# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ contrast, lib }:
contrast.initializer.overrideAttrs (old: {
  meta = (old.meta or { }) // {
    platforms = lib.platforms.linux;
  };
})
