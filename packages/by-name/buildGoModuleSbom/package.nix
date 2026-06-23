# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Builds a CycloneDX SBOM for a buildGoModule package by analysing its module
# with cyclonedx-gomod against the package's own vendored dependencies. Set
#
#   passthru.bombonVendoredSbom = buildGoModuleSbom { package = finalAttrs.finalPackage; };
#
# on a Go package; the top-level sbom package (bombon) collects it automatically.
# The output is a directory of CycloneDX files (one per main), as bombon expects
# for a bombonVendoredSbom, emitted at spec 1.5 (the version bombon parses).
#
# The package MUST set proxyVendor = true: cyclonedx-gomod runs `go mod graph`,
# which needs a module proxy (the full graph), so a -mod=vendor directory is not
# enough. .goModules is then used as an offline file:// GOPROXY.

{
  lib,
  runCommand,
  cyclonedx-gomod,
  jq,
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
  "${pname}-sbom"
  {
    nativeBuildInputs = [
      cyclonedx-gomod
      jq
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

    # One CycloneDX file per main; bombon reads the directory and dedupes
    # components across them, so no merge or bom-ref fixup is needed here.
    mkdir -p "$out"
    for main in ${lib.escapeShellArgs mains}; do
      out_name=$(echo "$main" | tr '/' '-')
      [ "$out_name" = "." ] && out_name=${pname}
      cyclonedx-gomod app \
        -json \
        -licenses \
        -packages \
        -output-version 1.5 \
        -main "$main" \
        -output "$out/$out_name.cdx.json" \
        .

      # cyclonedx-gomod records detected licenses as `evidence.licenses` (it
      # treats detection as non-authoritative). TR-03183-2 requires the licence
      # as the concluded component `licenses` field, so promote the evidence into
      # it (keeping evidence) for every component that has one.
      out_file="$out/$out_name.cdx.json"
      jq '
        def promote:
          (if (.evidence.licenses // [] | length) > 0 then .licenses = .evidence.licenses else . end)
          | (if has("components") then .components |= map(promote) else . end);
        .metadata.component |= promote
        | .components |= map(promote)
      ' "$out_file" > "$out_file.tmp" && mv "$out_file.tmp" "$out_file"
    done
  ''
