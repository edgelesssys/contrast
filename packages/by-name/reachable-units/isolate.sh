#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Usage: isolate COMMAND...
#
# Runs COMMAND in a new root that only contains /nix, $TMPDIR, /proc, /dev/null, an empty cgroup2 hierarchy, and empty /run and /tmp directories, using unprivileged namespaces.
# This way, files and mounts of the host can't influence the result of reachable-units, and the result is the same in and outside of the Nix build sandbox.
#
# COMMAND runs as an unprivileged user, as systemd refuses to run test mode as root. It can still create nested user namespaces, e.g. to become root again.

tmp=${TMPDIR:-/tmp}
root=$(mktemp -d)
trap 'rmdir "$root"' EXIT

# shellcheck disable=SC2016
unshare --user --map-root-user --mount --cgroup -- sh -euc '
  root=$1 tmp=$2
  shift 2
  mount -t tmpfs none "$root"
  mkdir -p "$root/nix" "$root$tmp" "$root/proc" "$root/sys/fs/cgroup" "$root/dev" "$root/run" "$root/tmp"
  mount --rbind /nix "$root/nix"
  mount --rbind "$tmp" "$root$tmp"
  mount --rbind /proc "$root/proc"
  mount -t cgroup2 none "$root/sys/fs/cgroup"
  touch "$root/dev/null"
  mount --bind /dev/null "$root/dev/null"
  # Use pivot_root instead of chroot, as chrooted processes can not create user namespaces.
  mkdir "$root/oldroot"
  cd "$root"
  pivot_root . oldroot
  umount --lazy /oldroot
  rmdir /oldroot
  exec unshare --user --map-user=1000 --map-group=1000 -- "$@"
' isolate "$root" "$tmp" "$@"
