# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  writers,
  runCommand,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule rec {
  pname = "nydus-snapshotter";
  version = "0.13.13";

  src = fetchFromGitHub {
    owner = "containerd";
    repo = "nydus-snapshotter";
    rev = "v${version}";
    hash = "sha256-InUBFTGBQR7LAv4rs9Smcdr7+iD1EHZr/JZ0M3pYK1Q=";
  };

  vendorHash = "sha256-Lb0j+VnjDyWmi09CHa8P48psVeZHUxI5I++ZaIV4Yog=";
  proxyVendor = true;

  subPackages = [
    "cmd/containerd-nydus-grpc"
    "cmd/nydus-overlayfs"
  ];

  CGO_ENABLED = "0";

  ldflags = [
    "-s"
    "-X github.com/containerd/nydus-snapshotter/version.Version=${version}"
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
}
