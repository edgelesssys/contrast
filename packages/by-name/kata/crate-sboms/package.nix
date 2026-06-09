# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runCommand,
  cargo,
  cargo-cyclonedx,
  rustc,
  rustPlatform,
  runtime,
  sbom-generator,
}:

# kata-agent, genpolicy and runtime-rs are all members of the same Kata Cargo
# workspace and share a single source tree and vendored dependencies, so one
# cargo-cyclonedx run over the workspace produces a per-member SBOM. We keep
# only the three crates we ship; each Rust package exposes its slice as
# passthru.sbom. The bom-refs of workspace-local crates contain the absolute
# manifest path, so we rewrite it to a stable, location-independent id.
runCommand "kata-crate-sboms"
  {
    nativeBuildInputs = [
      cargo
      cargo-cyclonedx
      rustc
      sbom-generator
    ];
    vendorDir = rustPlatform.importCargoLock {
      lockFile = "${runtime.src}/Cargo.lock";
      outputHashes = runtime.cargoOutputHashes;
    };
  }
  ''
    export HOME=$TMPDIR
    cp -r --no-preserve=mode,ownership ${runtime.src} kata-src
    katasrc="$PWD/kata-src"
    export CARGO_HOME="$TMPDIR/cargo-home"
    export CARGO_NET_OFFLINE=true
    mkdir -p "$CARGO_HOME"
    sed "s|directory = \"cargo-vendor-dir\"|directory = \"$vendorDir\"|" \
      "$vendorDir/.cargo/config.toml" >"$CARGO_HOME/config.toml"

    pushd "$katasrc" >/dev/null
    cargo cyclonedx \
      --manifest-path Cargo.toml \
      --format json \
      --describe crate \
      --spec-version 1.5 \
      -qq
    popd >/dev/null

    mkdir -p "$out"
    for member in \
      src/agent/kata-agent \
      src/tools/genpolicy/genpolicy \
      src/runtime-rs/runtime-rs; do
      name="''${member##*/}"
      dst="$out/$name.cdx.json"
      sed "s|$katasrc|kata-containers|g" "$katasrc/$member.cdx.json" >"$dst"
      sbom-generator fix-bomrefs "$dst"
    done
  ''
