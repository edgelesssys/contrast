# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runCommand,
  closureInfo,
  cyclonedx-gomod,
  cyclonedx-cli,
  contrast,
  reference-values,
  sbom-generator,
  go,
  git,
  cacert,
}:

let
  closure = closureInfo {
    rootPaths = [
      contrast.coordinator
      contrast.initializer
    ];
  };
in

runCommand "contrast-sbom.cdx.json"
  {
    nativeBuildInputs = [
      cyclonedx-gomod
      cyclonedx-cli
      sbom-generator
      go
      git
      cacert
    ];
    passthru = { inherit contrast closure; };
  }
  ''
    export HOME=$TMPDIR
    export XDG_CACHE_HOME=$TMPDIR/xdg-cache

    cp -r --no-preserve=mode,ownership ${contrast.src} src
    pushd src >/dev/null
    install -D ${reference-values} internal/manifest/assets/reference-values.json
    git init -q -b main
    git -c user.email=sbom@contrast -c user.name=sbom add -A
    git -c user.email=sbom@contrast -c user.name=sbom \
      -c commit.gpgsign=false commit -q -m sbom

    export GOPROXY="file://${contrast.goModules}"
    export GOSUMDB=off
    export GOMODCACHE=$TMPDIR/gomodcache
    export GOCACHE=$TMPDIR/gocache
    export GOFLAGS="-tags=contrast_unstable_api"
    export CGO_ENABLED=0
    for bin in coordinator initializer; do
      cyclonedx-gomod app \
        -json \
        -licenses \
        -packages \
        -main "$bin" \
        -output "$TMPDIR/go.$bin.cdx.json" \
        .
      sbom-generator fix-bomrefs "$TMPDIR/go.$bin.cdx.json"
    done
    popd >/dev/null

    sbom-generator closure \
      --store-paths ${closure}/store-paths \
      --version ${contrast.version} \
      --output "$TMPDIR/nix.closure.cdx.json"

    cyclonedx merge \
      --hierarchical \
      --name contrast \
      --group io.edgeless \
      --version ${contrast.version} \
      --input-files \
        "$TMPDIR/nix.closure.cdx.json" \
        "$TMPDIR/go.coordinator.cdx.json" \
        "$TMPDIR/go.initializer.cdx.json" \
      --output-file "$out" \
      --output-format json
  ''
