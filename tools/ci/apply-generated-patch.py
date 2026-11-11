# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

"""Apply an untrusted generated patch without executing its contents."""

import argparse
import pathlib
import subprocess


def git(*args):
    return subprocess.check_output(["git", *args])


def apply(patch, only_paths):
    if git("status", "--porcelain", "--untracked-files=no"):
        raise ValueError("target checkout must be clean")
    try:
        # Stage first so even a rejected patch cannot plant files in the checkout.
        git("apply", "--cached", "--whitespace=nowarn", str(patch))
        paths = git("diff", "--cached", "--name-only", "-z").split(b"\0")
        for raw_path in filter(None, paths):
            path = raw_path.decode("utf-8")
            parts = pathlib.PurePosixPath(path).parts
            if (
                parts[0] == ".github"
                or parts[:2] == ("tools", "ci")
                or ".gitattributes" in parts
                or (only_paths and path not in only_paths)
            ):
                raise ValueError(f"patch may not change {path}")
            entry = git("ls-files", "--stage", "-z", "--", path)
            if entry and entry.split(b" ", 1)[0] not in (b"100644", b"100755"):
                raise ValueError(f"patch may only write regular files: {path}")
    except Exception:
        git("read-tree", "HEAD")
        raise
    git("checkout-index", "--all", "--force")
    # checkout-index doesn't remove deleted paths.
    for raw_path in filter(None, git("diff", "--cached", "--diff-filter=D", "--name-only", "-z").split(b"\0")):
        pathlib.Path(raw_path.decode("utf-8")).unlink()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("patch", type=pathlib.Path)
    parser.add_argument("--only-path", action="append", default=[])
    args = parser.parse_args()
    apply(args.patch.resolve(), args.only_path)
