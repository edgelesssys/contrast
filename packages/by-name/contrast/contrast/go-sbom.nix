# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# CycloneDX SBOM for the contrast Go binaries (coordinator and initializer),
# built offline from the module's vendored dependencies. Exposed as
# contrast.contrast.passthru.sbom and collected by the top-level sbom package.
{
  lib,
  runCommand,
  cyclonedx-gomod,
  cyclonedx-cli,
  sbom-generator,
  reference-values,
  go,
  git,
  cacert,

  src,
  goModules,
  version,
  tags,
}:

runCommand "contrast-go.cdx.json"
  {
    nativeBuildInputs = [
      cyclonedx-gomod
      cyclonedx-cli
      sbom-generator
      go
      git
      cacert
    ];
  }
  ''
    export HOME=$TMPDIR
    export XDG_CACHE_HOME=$TMPDIR/xdg-cache

    cp -r --no-preserve=mode,ownership ${src} src
    pushd src >/dev/null
    install -D ${reference-values} internal/manifest/assets/reference-values.json
    git init -q -b main
    git -c user.email=sbom@contrast -c user.name=sbom add -A
    git -c user.email=sbom@contrast -c user.name=sbom \
      -c commit.gpgsign=false commit -q -m sbom

    export GOPROXY="file://${goModules}"
    export GOSUMDB=off
    export GOMODCACHE=$TMPDIR/gomodcache
    export GOCACHE=$TMPDIR/gocache
    export GOFLAGS="-tags=${lib.concatStringsSep "," tags}"
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

    # Flat merge (not --hierarchical): hierarchical merges emit a schema-invalid
    # dependencies graph for inputs that share modules, as coordinator and
    # initializer do. A flat merge dedupes the shared modules and stays valid.
    cyclonedx merge \
      --name contrast-go \
      --group io.edgeless \
      --version ${version} \
      --input-files \
        "$TMPDIR/go.coordinator.cdx.json" \
        "$TMPDIR/go.initializer.cdx.json" \
      --output-file "$out" \
      --output-format json
  ''
