# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
  kata,

  runCommand,
  skopeo,
  containers,
  writeText,
}:

let
  pullOciImage =
    {
      imageName,
      imageDigest,
      hash,
    }:
    runCommand "pull-oci-image-${imageName}"
      {
        outputHash = hash;
        outputHashAlgo = "sha256";
        outputHashMode = "recursive";
        nativeBuildInputs = [ skopeo ];
      }
      ''
        skopeo copy \
          --insecure-policy \
          --tmpdir=$TMPDIR \
          --all \
          docker://${imageName}@${imageDigest} oci:$out
      '';
  busybox = pullOciImage {
    imageName = "busybox";
    imageDigest = "sha256:dc2d74b28e4cf8984fa52af1f39bc7c3d9c73760b41a74d629f5d11b1ab28616";
    hash = "sha256-QQmWRMR1a5i/BnUCn3I5DFXeH3dwfmiNVIQkEDV9hYs=";
  };
  pause = pullOciImage {
    imageName = "ghcr.io/edgelesssys/kubernetes/pause";
    imageDigest = "sha256:b4b669f27933146227c9180398f99d8b3100637e4a0a1ccf804f8b12f4b9b8df";
    hash = "sha256-Y7fNAjPe9/RIHVSJ/EmreJ/cnhizzs22A++SpOI3bAc=";
  };
  initializer = runCommand "initializer-oci" { } ''
    # Resolve symlinks in the OCI directory, because go-containerregistry doesn't accept symlinked blobs.
    cp -rL ${containers.initializer} $out
  '';
  imagesJson = writeText "images.json" (
    builtins.toJSON {
      busybox = {
        path = "${busybox}";
        ref = "busybox@sha256:0000000000000000000000000000000000000000000000000000000000000000";
      };
      pause = {
        path = "${pause}";
        ref = "ghcr.io/edgelesssys/kubernetes/pause:3.6";
      };
      initializer = {
        path = "${initializer}";
        ref = "ghcr.io/edgelesssys/contrast/initializer:latest";
      };
    }
  );
in

buildGoModule (_finalAttrs: {
  pname = "policy-test";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (fileset.difference (path.append root "policy-test") (path.append root "policy-test/testdata"))
        (path.append root "cli")
        (path.append root "internal")
        (path.append root "sdk")
        (path.append root "apitypes")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-OSv0Xg4F2td+Maq+ALfnoa6iX9OyjG5P/KTAsMerOGo=";

  modRoot = "policy-test";

  preConfigure = ''
    install -D ${imagesJson} policy-test/assets/images.json
    # The default file under cli/cmd/assets/genpolicy-settings-kata.json will still contain a placeholder
    # but this doesn't matter because we will always call genpolicy with a custom settings file.
    install -D ${kata.genpolicy.settings-dev}/genpolicy-settings.json policy-test/assets/genpolicy-settings-kata.json
    # Move postConfigure here, because the configurePhase already cd's into modRoot.
    ${contrast.cli.postConfigure}
  '';

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
  ];

  tags = [ "contrast_unstable_api" ];

  meta = lib.contrast.ourMeta { mainProgram = "policy-test"; };
})
