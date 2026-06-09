# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Builds a CycloneDX SBOM for a buildGoModule package by analysing its module
# with cyclonedx-gomod against the package's own vendored dependencies. Set
#
#   passthru.sbom = buildGoModuleSbom { package = finalAttrs.finalPackage; };
#
# on a Go package and the top-level sbom package collects it automatically.
#
# The package MUST set proxyVendor = true: cyclonedx-gomod runs `go mod graph`,
# which needs a module proxy (the full graph), so a -mod=vendor directory is not
# enough. .goModules is then used as an offline file:// GOPROXY.

{
  lib,
  runCommand,
  cyclonedx-gomod,
  cyclonedx-cli,
  sbom-generator,
  go,
  git,
  cacert,
}:

{
  package,
  # Main package directories (relative to the module root) to emit an app SBOM
  # for. Defaults to the package's subPackages, or the module root.
  mains ? package.subPackages or [ "." ],
  # Shell run in the module root before analysis, e.g. to install generated or
  # embedded assets the module needs to load (go list fails otherwise). Defaults
  # to the package's own postConfigure, so embeds such as reference-values.json
  # are put in place exactly as the real build does it.
  preAnalyze ? package.postConfigure or "",
  pname ? package.pname,
}:

let
  # When the package builds from a subdirectory (sourceRoot), analyse there.
  moduleRoot =
    if package ? sourceRoot then lib.removePrefix "${package.src.name}/" package.sourceRoot else ".";
  tags = package.tags or [ ];
  tagsFlag = lib.optionalString (tags != [ ]) "-tags=${lib.concatStringsSep "," tags}";
in

lib.throwIfNot (package.proxyVendor or false)
  "buildGoModuleSbom: ${pname} must set proxyVendor = true; cyclonedx-gomod needs a module proxy (go mod graph), so a vendor directory is not sufficient"

  runCommand
  "${pname}.cdx.json"
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

    cp -r --no-preserve=mode,ownership ${package.src} src
    # cyclonedx-gomod derives the main module version from VCS, and modules with
    # a local `replace` need the replaced tree under VCS too, so init at the repo
    # root rather than the (possibly nested) module directory.
    pushd src >/dev/null
    git init -q -b main
    git -c user.email=sbom@contrast -c user.name=sbom add -A
    git -c user.email=sbom@contrast -c user.name=sbom \
      -c commit.gpgsign=false commit -q -m sbom
    popd >/dev/null

    pushd src/${moduleRoot} >/dev/null
    ${preAnalyze}

    export GOPROXY="file://${package.goModules}"
    export GOSUMDB=off
    export GOMODCACHE=$TMPDIR/gomodcache
    export GOCACHE=$TMPDIR/gocache
    ${lib.optionalString (tagsFlag != "") ''export GOFLAGS="${tagsFlag}"''}
    export CGO_ENABLED=0

    sbomdir=$TMPDIR/sboms
    mkdir -p "$sbomdir"
    for main in ${lib.escapeShellArgs mains}; do
      out_name=$(echo "$main" | tr '/' '-')
      [ "$out_name" = "." ] && out_name=${pname}
      cyclonedx-gomod app \
        -json \
        -licenses \
        -packages \
        -main "$main" \
        -output "$sbomdir/$out_name.cdx.json" \
        .
      sbom-generator fix-bomrefs "$sbomdir/$out_name.cdx.json"
    done
    popd >/dev/null

    # Flat merge: hierarchical merges emit a schema-invalid dependencies graph
    # for modules that share packages (and it normalises a single input too).
    cyclonedx merge \
      --name ${pname} \
      --group io.edgeless \
      --version ${package.version} \
      --input-files "$sbomdir"/*.cdx.json \
      --output-file "$out" \
      --output-format json
  ''
