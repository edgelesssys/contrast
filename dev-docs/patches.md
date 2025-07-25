# Working with patches

## Patches in Nix

There are two common ways to apply patches in Nix:

- Use the `patches` attribute of a builder
  ([manual](https://nixos.org/manual/nixpkgs/unstable/#var-stdenv-patches)):

  ```nix
  stdenv.mkDerivation {
    # ...
    patches = [
        ./0001-first-change.patch
        ./0002-another-change.patch
    ]
    # ...
  }
  ```
  This is most commonly used and can be encountered nearly everywhere upstream
  in nixpkgs.

- Use the `applyPatches` function
  ([noogle](https://noogle.dev/f/pkgs/applyPatches)) to patch an existing source
  and derive a new source:

  ```nix
  stdenv.mkDerivation {
    # ...
    src = applyPatches {
        src = fetchFromGuiHub { /* ... */ }
    patches = [
        ./0001-first-change.patch
        ./0002-another-change.patch
    ]
    # ...
  }
  ```

  This is useful in situations where the patches sources should also be used by
  other consumers, as the patched source can inherited, but `patches` aren't
  part of the output set.

Patches can either be consumed from a local file or pulled from remote:

```nix
patches = [
  (fetchpatch {
    url = "https://github.com/...";
    stripLen = 2; # Sometimes the patch strip level must be configured
    hash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
  })
];
```

If there aren't any changes needed to an upstream patch, it's good practice to
fetch the patch from remote. The patch should come from a stable ref in this
case, like from a persistent upstream branch or an open pull request. If the
changes to upstream patches are needed or patches aren't available upstream, the
_full patch set_ should be vendored (don't mix `fetchpatch` and local patches on
the same target), so it can easily be worked with.

## Patch set development flow

For a smooth development experience, we only use `git format-patch` format for
patches. This enables us to apply the changes to the source, work on it and sync
back.

Given a package with some existing patches in `pkgDir` (for example
`packages/by-name/contrast`).

Clone the source and checkout the `rev` currently used in the package `src` (the
commit or tag):

```sh
git clone $url
git checkout $rev
```

Apply the existing patch set:

```sh
git am --no-3way $pkgDir/0*.patch
```

This will apply and commit each patch on top of `rev`. Some directories contain
patches that aren't meant to be applied to the source, those are excluded by the
`0` prefix. The `--no-3way` flag will abort application of unclean patches. If
the existing patches can't be applied without a three-way merge, you can pass
`-3` instead. This situation should then be resolved in a separate commit.

You can then place new commits on top or modify existing commits. Remember to
keep the git history clean.

When updating a package, you might need to rebase the current patch set.

When done, recreate the patch set:

```sh
git format-patch -N --no-signature --zero-commit --full-index -o $pkgDir $rev
```

Don't forget to `git add` patches you just added and to `git rm` patches you
removed or renamed.

The `static.yml` workflow ensures that patches can be reapplied cleanly. If this
workflow fails, applying the rendered diff should be sufficient to appease it.

# Patch documentation conventions

Patches need thorough documentation. Each reference of a patch must have a
comment:

```nix
patches = [
  # Document things about the patch here
  ./0001-first-change.patch
]
```

Notice that this is in addition to writing a sensible commit message in case you
create a commit specifically for a patch. In many cases, commit message of a
patch will come from original patch author, so all the meta goes where the patch
is referenced.

The comment should answer the following questions:

- Why is the patch needed?
- Is the patch already merged? -> Make a note that the patch can be removed on
  next update.
- Is the patch part of an open upstream PR? -> Link to the upstream PR.
- If it's an upstream patch, was the patch modified? If yes, what was changed
  and why?
- If the patch has no upstream PR, why can't we upstream the change?
