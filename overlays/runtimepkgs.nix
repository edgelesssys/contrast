# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

final: prev:

if prev.stdenv.hostPlatform.system == "x86_64-linux" then
  { }
else
  {
    contrastPkgs = prev.contrastPkgs.overrideScope (
      _cFinal: cPrev: {
        # genpolicy needs to be built natively since macOS doesn't support static binaries.
        contrastPkgsStatic = final.runtimePkgs.contrastPkgsStatic.overrideScope (
          _: _: {
            kata = final.runtimePkgs.contrastPkgsStatic.kata.overrideScope (
              _: _: {
                # On darwin, strip dylib load commands the linker leaves
                # behind but no symbol references, and assert the binary
                # has no /nix/store dylib paths in its load commands.
                # The instance prompting this is libc 0.2.x's
                # `#[link(name = "iconv")]` on the apple FFI bindings:
                # rustc emits `-liconv` for every Rust binary that links
                # libc, which the macOS linker resolves to a build-host
                # nix-store path inside LC_LOAD_DYLIB. See
                # https://github.com/rust-lang/libc/issues/2870.
                #
                # TODO(sespiros): drop once kata's Cargo.lock pins a libc
                # that no longer emits the directive (libc 1.0 milestone).
                genpolicy = cPrev.kata.genpolicy.overrideAttrs (oldAttrs: {
                  env =
                    (oldAttrs.env or { })
                    // prev.lib.optionalAttrs prev.stdenv.hostPlatform.isDarwin {
                      RUSTFLAGS = "-C link-arg=-Wl,-dead_strip_dylibs";
                    };

                  # Regression guard: assert the darwin binary has no
                  # /nix/store dylib load commands. `otool -L`'s first line
                  # is the binary's own path (itself under /nix/store), so
                  # skip it. Catches anything new dyld-linked from nixpkgs,
                  # not just iconv.
                  postInstall =
                    (oldAttrs.postInstall or "")
                    + prev.lib.optionalString prev.stdenv.hostPlatform.isDarwin ''
                      nixstore_deps=$(otool -L "$out/bin/genpolicy" | tail -n +2 | grep '/nix/store' || true)
                      if [[ -n "$nixstore_deps" ]]; then
                        echo "error: genpolicy has /nix/store paths in its dyld load commands:" >&2
                        echo "$nixstore_deps" >&2
                        exit 1
                      fi
                    '';
                });
              }
            );
          }
        );

        kata = cPrev.kata.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.kata)
              agent
              image
              kernel-uvm
              runtime
              runtime-rs
              calculateSnpLaunchDigest
              calculateTdxLaunchDigests
              ;
          }
        );

        contrast = cPrev.contrast.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.contrast)
              coordinator
              initializer
              node-installer-image
              nodeinstaller
              reference-values
              snp-launch-digests
              ;
          }
        );

        inherit (final.runtimePkgs)
          badaml-payload
          debugshell
          service-mesh
          k8s-log-collector
          boot-image
          boot-microvm
          qemu-cc
          pause-bundle
          OVMF-SNP
          OVMF-TDX
          ;

        scripts = cPrev.scripts.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.scripts)
              cleanup-bare-metal
              cleanup-images
              cleanup-containerd
              nix-gc
              upgrade-gpu-operator
              sev-snp-measure-consistency
              ;
          }
        );

        inherit (final.runtimePkgs) containers;
      }
    );
  }
