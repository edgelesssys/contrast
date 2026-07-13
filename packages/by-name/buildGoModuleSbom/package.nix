# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
  mains ? package.subPackages or [ "." ],
  preAnalyze ? package.postConfigure or "",
  pname ? package.pname,
}:

let
  moduleRoot =
    if package ? sourceRoot then lib.removePrefix "${package.src.name}/" package.sourceRoot else ".";
  tags = package.tags or [ ];
  tagsFlag = lib.optionalString (tags != [ ]) "-tags=${lib.concatStringsSep "," tags}";
in

lib.throwIfNot (package.proxyVendor or false)
  "buildGoModuleSbom: ${pname} must set proxyVendor = true; cyclonedx-gomod needs a module proxy / go mod graph, a vendor directory is insufficient"

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
  # bash
  ''
    export HOME=$TMPDIR
    export XDG_CACHE_HOME=$TMPDIR/xdg-cache

    cp -r --no-preserve=mode,ownership ${package.src} src
    pushd src >/dev/null
    git init -q -b main
    git -c user.email=sbom@contrast -c user.name=sbom add -A
    git -c user.email=sbom@contrast -c user.name=sbom \
      -c commit.gpgsign=false commit -q -m sbom
    # Tag HEAD with the real release version so cyclonedx-gomod reports it instead of a v0.0.0-<commit> pseudo-version
    git -c user.email=sbom@contrast -c user.name=sbom tag -a -m sbom "v${lib.fileContents ../../../version.txt}"
    popd >/dev/null

    pushd src/${moduleRoot} >/dev/null
    ${preAnalyze}

    export GOPROXY="file://${package.goModules}"
    export GOSUMDB=off
    export GOMODCACHE=$TMPDIR/gomodcache
    export GOCACHE=$TMPDIR/gocache
    ${lib.optionalString (tagsFlag != "") ''export GOFLAGS="${tagsFlag}"''}
    export CGO_ENABLED=0

    mkdir -p "$out"
    for main in ${lib.escapeShellArgs mains}; do
      outName=$(echo "$main" | tr '/' '-')
      [ "$outName" = "." ] && outName=${pname}

      cyclonedx-gomod app \
        -json \
        -licenses \
        -output-version 1.5 \
        -main "$main" \
        -output "$out/$outName.cdx.json" \
        .

      # cyclonedx-gomod's quirks:
      #  - promote `evidence.licenses` to the concluded `licenses`
      #  - derive source-code `vcs` reference from the module path
      outFile="$out/$outName.cdx.json"
      jq '
        def promote:
          if (.evidence.licenses // [] | length) > 0 then .licenses = .evidence.licenses else . end;
        def with_vcs:
          if (.externalReferences // []) | any(.type == "vcs") then .
          else .externalReferences = ((.externalReferences // [])
            + [ { type: "vcs", url: ("https://" + (.name | sub("/v[0-9]+$"; ""))) } ]) end;
        .metadata.component |= promote
        | .components |= map(promote | with_vcs)
      ' "$outFile" > "$outFile.tmp" && mv "$outFile.tmp" "$outFile"
    done
  ''
