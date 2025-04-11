# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  writers,
  runCommand,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule (finalAttrs: {
  pname = "nydus-snapshotter";
  version = "0.13.14";

  src = fetchFromGitHub {
    owner = "containerd";
    repo = "nydus-snapshotter";
    rev = "v${finalAttrs.version}";
    hash = "sha256-DlBYZtgYl200ZEO2tbSte5bGFIJw6UWeRbMzpe2gp2U=";
  };

  patches = [
    ./0001-disable-database-flock-timeout.patch
    ./0002-snapshotter-add-flag-to-specify-nydus-overlayfs-path.patch
  ];

  vendorHash = "sha256-sbdlxmuqN72YbEGv4BPYsTBrowX8YtsFDuHf1SdJ4tw=";
  proxyVendor = true;

  subPackages = [
    "cmd/containerd-nydus-grpc"
    "cmd/nydus-overlayfs"
  ];

  env.CGO_ENABLED = "0";

  ldflags = [
    "-s"
    "-X github.com/containerd/nydus-snapshotter/version.Version=${finalAttrs.version}"
  ];

  passthru = {
    # Based on https://github.com/confidential-containers/operator/blob/6b249fd671a683120a9aac860b953fe7f0e40a1b/install/pre-install-payload/remote-snapshotter/nydus-snapshotter/config-coco-guest-pulling.toml
    config =
      let
        configFile = writers.writeTOML "nydus-snapshotter-config" {
          version = 1;
          daemon.fs_driver = "proxy";
          snapshot.enable_kata_volume = true;
        };
      in
      runCommand "config-coco-guest-pulling.toml" { } ''
        mkdir -p $out/share/nydus-snapshotter
        ln -s ${configFile} $out/share/nydus-snapshotter/config-coco-guest-pulling.toml
      '';
  };

  meta = with lib; {
    description = "A containerd snapshotter with data deduplication and lazy loading in P2P fashion";
    homepage = "https://github.com/containerd/nydus-snapshotter";
    license = licenses.asl20;
    mainProgram = "nydus-snapshotter";
  };
})
