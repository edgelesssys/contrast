# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

import pathlib
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name("apply-generated-patch.py")


class ApplyGeneratedPatchTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = pathlib.Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "Test")
        self.git("config", "user.email", "test@example.invalid")
        (self.repo / "generated.txt").write_text("before\n")
        self.git("add", "generated.txt")
        self.git("commit", "-qm", "fixture")

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.repo), *args])

    def patch(self, path, contents):
        target = self.repo / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(contents)
        self.git("add", "--", path)
        patch = self.git("diff", "--cached", "--binary")
        self.git("reset", "--hard", "HEAD")
        artifact = self.repo / "changes.patch"
        artifact.write_bytes(patch)
        return artifact

    def apply(self, patch, *args):
        return subprocess.run(
            ["python3", str(SCRIPT), str(patch), *args],
            cwd=self.repo, capture_output=True, text=True,
        )

    def test_applies_binary_and_text_changes(self):
        patch = self.patch("generated.txt", b"after\x00\xff\n")
        result = self.apply(patch)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.repo / "generated.txt").read_bytes(), b"after\x00\xff\n")

    def test_rejects_workflow_and_publisher_changes(self):
        for path in (".github/workflows/publish.yml", "tools/ci/publisher.py", ".gitattributes"):
            with self.subTest(path=path):
                patch = self.patch(path, b"untrusted\n")
                result = self.apply(patch)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.git("diff", "--cached"), b"")

    def test_rejects_symlinks(self):
        target = self.repo / "generated-link"
        target.symlink_to("generated.txt")
        self.git("add", "generated-link")
        patch = self.git("diff", "--cached", "--binary")
        self.git("reset", "--hard", "HEAD")
        artifact = self.repo / "changes.patch"
        artifact.write_bytes(patch)
        self.assertNotEqual(self.apply(artifact).returncode, 0)
        self.assertFalse(target.exists())

    def test_restricts_updater_to_expected_path(self):
        patch = self.patch("generated.txt", b"after\n")
        result = self.apply(patch, "--only-path", "dev-docs/nvidia/rim-ids.json")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((self.repo / "generated.txt").read_text(), "before\n")

    def test_rejects_stale_patch(self):
        patch = self.patch("generated.txt", b"after\n")
        (self.repo / "generated.txt").write_text("new head\n")
        self.git("add", "generated.txt")
        self.git("commit", "-qm", "advance")
        self.assertNotEqual(self.apply(patch).returncode, 0)
        self.assertEqual((self.repo / "generated.txt").read_text(), "new head\n")


if __name__ == "__main__":
    unittest.main()
