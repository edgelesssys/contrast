# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  stdenvNoCC,
  fetchFromGitHub,
}:

let
  # The following RBAC definitions are normally installed by the deploy/kubernetes-latest/deploy.sh
  # script. Repo tags are dynamically derived from the resources in
  # deploy/kubernetes-latest-test/hostpath/, which is a bit awkward to reproduce here. When
  # upgrading the csi-driver-host-path repository, make sure to take a look at the changes in that
  # directory and upgrade all of the changed RBAC repos.
  attacherRBAC = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kubernetes-csi/external-attacher/v4.8.0/deploy/kubernetes/rbac.yaml";
    sha256 = "sha256:0rm7j1y1viyymj1kmfdv5lz2dqj33pkn6v721iy955h4iccvaf1s";
  };
  healthRBAC = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kubernetes-csi/external-health-monitor/v0.16.0/deploy/kubernetes/external-health-monitor-controller/rbac.yaml";
    sha256 = "sha256:09yiwbgkl8jlrrsm5aqs7hvxmn5gj4jy5axc6wkjw9csvag5j19j";
  };
  provisionerRBAC = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kubernetes-csi/external-provisioner/v5.2.0/deploy/kubernetes/rbac.yaml";
    sha256 = "sha256:01vmgzj42vb5hhx97rzsamh8f128bn2x6pksbd4yi6f7vf4zag3h";
  };
  resizerRBAC = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kubernetes-csi/external-resizer/v1.13.1/deploy/kubernetes/rbac.yaml";
    sha256 = "sha256:1rxb0rdqqsw2mfc419b36sy99w04dzsgw28zlyrpgi3d5vcxjsdw";
  };
  snapshotterRBAC = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v8.2.0/deploy/kubernetes/csi-snapshotter/rbac-csi-snapshotter.yaml";
    sha256 = "sha256:164wvh5i017wb8223fbp6z0x86a3xmznaks9d2canzzi29jgcryy";
  };
in
stdenvNoCC.mkDerivation {
  pname = "csi-driver-host-path";
  version = "v1.18.0-pre.2025-11-18";

  src = fetchFromGitHub {
    owner = "kubernetes-csi";
    repo = "csi-driver-host-path";
    rev = "69142b68de86efed13e615c2ed9c98b62b5234a2";
    hash = "sha256-dr+oQZfTNN5+U0rViiebYvsmwkqA5c8sV+5IAk3oR5k=";
  };

  dontBuild = true;

  installPhase = ''
    runHook preInstall

    install -t $out -D $src/deploy/kubernetes-latest/hostpath/csi-hostpath-{driverinfo,plugin}.yaml

    install -D ${./csi-storageclass.yaml} $out/csi-storageclass.yaml
    install -D ${./kustomization.yaml} $out/kustomization.yaml
    install -D ${./namespace.yaml} $out/namespace.yaml

    install -D ${attacherRBAC} $out/csi-attacher-rbac.yaml
    install -D ${healthRBAC} $out/csi-health-rbac.yaml
    install -D ${provisionerRBAC} $out/csi-provisioner-rbac.yaml
    install -D ${resizerRBAC} $out/csi-resizer-rbac.yaml
    install -D ${snapshotterRBAC} $out/csi-snapshotter-rbac.yaml

    runHook postInstall
  '';

  meta = {
    homepage = "https://github.com/kubernetes-csi/csi-driver-host-path";
    license = with lib.licenses; [
      asl20
    ];
  };
}
