"""Offline regression tests: no Ceph or Kubernetes connection is made."""
import contextlib
import hashlib
import importlib.util
import io
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("ceph_worker", Path(__file__).with_name("worker.py"))
worker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(worker)


class BackupTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = patch.object(worker, "ROOT", Path(self.temp.name))
        root.start()
        self.addCleanup(root.stop)
        self.output = contextlib.redirect_stdout(io.StringIO())
        self.output.__enter__()
        self.addCleanup(self.output.__exit__, None, None, None)

    def write_backup(self, backup_id, parent="", **changes):
        content = ("diff for " + backup_id).encode()
        value = dict(version=1, id=backup_id, pool="rbd", image="source", fsid="cluster-1",
                     snapshot="snap-" + backup_id, fromSnapshot="snap-" + parent if parent else "",
                     parent=parent, size=1048576, bytes=len(content), sha256=hashlib.sha256(content).hexdigest())
        value.update(changes)
        (worker.ROOT / (backup_id + ".diff")).write_bytes(content)
        (worker.ROOT / (backup_id + ".json")).write_text(json.dumps(value))
        return value

    def restore(self, backup_id="inc"):
        worker.backup_task("backup-restore", dict(backupId=backup_id, targetPool="dest", targetImage="new"), "restore-task")

    def test_restore_validates_chain_then_imports_baseline_and_increment(self):
        self.write_backup("full")
        self.write_backup("inc", "full", size=2097152)
        commands = []

        def run(args, capture=False, check=True):
            commands.append(args)
            if args[:3] == ["rbd", "info", "dest/new"]:
                if len(commands) == 1:
                    return subprocess.CompletedProcess(args, 2, "", "No such file")
                return subprocess.CompletedProcess(args, 0, '{"size":2097152}', "")
            return subprocess.CompletedProcess(args, 0, "", "")

        with patch.object(worker, "run", side_effect=run):
            self.restore()
        self.assertEqual(commands[1], ["rbd", "create", "dest/new", "--size", "1"])
        self.assertEqual([Path(c[2]).name for c in commands if c[1] == "import-diff"], ["full.diff", "inc.diff"])
        self.assertTrue(all(c[-1] == "dest/new" for c in commands if c[1] == "import-diff"))
        self.assertTrue((worker.ROOT / "full.diff").exists())

    def test_restore_never_overwrites_existing_target(self):
        self.write_backup("full")
        with patch.object(worker, "run", return_value=subprocess.CompletedProcess([], 0, "{}", "")) as run:
            with self.assertRaisesRegex(RuntimeError, "已存在"):
                self.restore("full")
        self.assertEqual(run.call_count, 1)
        self.assertEqual(run.call_args[0][0][:2], ["rbd", "info"])

    def test_connection_failure_does_not_mean_target_absent(self):
        self.write_backup("full")
        with patch.object(worker, "run", return_value=subprocess.CompletedProcess([], 13, "", "access denied")) as run:
            with self.assertRaisesRegex(RuntimeError, "无法确认目标不存在"):
                self.restore("full")
        self.assertEqual(run.call_count, 1)

    def test_invalid_chain_fails_before_any_ceph_mutation(self):
        for problem in ("checksum", "missing-diff", "missing-parent", "cycle", "wrong-snapshot", "wrong-image", "wrong-fsid", "missing-fsid", "missing-parent-field", "boolean-size"):
            with self.subTest(problem=problem):
                self.write_backup("full")
                value = self.write_backup("inc", "full")
                if problem == "checksum":
                    path = worker.ROOT / "full.diff"
                    path.write_bytes(b"X" * path.stat().st_size)
                elif problem == "missing-diff":
                    (worker.ROOT / "full.diff").unlink()
                elif problem == "missing-parent":
                    value["parent"] = "gone"
                elif problem == "cycle":
                    value["parent"] = "inc"
                elif problem == "wrong-snapshot":
                    value["fromSnapshot"] = "wrong"
                elif problem == "wrong-image":
                    value["image"] = "other"
                elif problem == "wrong-fsid":
                    value["fsid"] = "other-cluster"
                elif problem == "missing-fsid":
                    del value["fsid"]
                elif problem == "missing-parent-field":
                    del value["parent"]
                elif problem == "boolean-size":
                    value["size"] = True
                (worker.ROOT / "inc.json").write_text(json.dumps(value))
                with patch.object(worker, "run") as run:
                    with self.assertRaises((ValueError, OSError)):
                        self.restore()
                    run.assert_not_called()

    def test_delete_preserves_parent_used_by_increment(self):
        self.write_backup("full")
        self.write_backup("inc", "full")
        with self.assertRaisesRegex(ValueError, "增量"):
            worker.backup_task("backup-delete", {"backupId": "full"}, "delete-task")
        self.assertTrue((worker.ROOT / "full.diff").exists())
        worker.backup_task("backup-delete", {"backupId": "inc"}, "delete-task")
        worker.backup_task("backup-delete", {"backupId": "full"}, "delete-task")
        self.assertEqual(list(worker.ROOT.glob("*.diff")), [])

    def test_repository_path_and_symlink_rejected(self):
        with self.assertRaises(ValueError):
            worker.safe_file("../outside", ".json")
        (worker.ROOT / "link.diff").symlink_to(worker.ROOT / "missing")
        with self.assertRaises(ValueError):
            worker.safe_file("link", ".diff")

    def test_increment_requires_same_cluster_and_existing_snapshot(self):
        self.write_backup("full", fsid="another-cluster")
        with patch.object(worker, "run", return_value=subprocess.CompletedProcess([], 0, "cluster-1\n", "")) as run:
            with self.assertRaisesRegex(ValueError, "同一镜像"):
                worker.backup_task("backup-incremental", dict(pool="rbd", image="source", baseBackup="full"), "next")
        self.assertEqual(run.call_count, 1)

    def test_failed_export_preserves_partial_without_publishing_backup(self):
        def run(args, capture=False, check=True):
            if args == ["ceph", "fsid"]:
                return subprocess.CompletedProcess(args, 0, "cluster-1", "")
            if args[:2] == ["rbd", "info"]:
                return subprocess.CompletedProcess(args, 0, '{"size":1048576}', "")
            if args[:2] == ["rbd", "export-diff"]:
                Path(args[-1]).write_bytes(b"incomplete")
                raise RuntimeError("export interrupted")
            return subprocess.CompletedProcess(args, 0, "", "")

        with patch.object(worker, "run", side_effect=run):
            with self.assertRaisesRegex(RuntimeError, "interrupted"):
                worker.backup_task("backup-full", dict(pool="rbd", image="source"), "failed")
        self.assertTrue((worker.ROOT / "failed.partial").exists())
        self.assertFalse((worker.ROOT / "failed.json").exists())
        self.assertFalse((worker.ROOT / "failed.diff").exists())

    def test_full_and_incremental_publish_verifiable_snapshot_chain(self):
        commands = []

        def run(args, capture=False, check=True):
            commands.append(args)
            if args == ["ceph", "fsid"]:
                return subprocess.CompletedProcess(args, 0, "cluster-1", "")
            if args[:2] == ["rbd", "info"]:
                return subprocess.CompletedProcess(args, 0, '{"size":1048576}', "")
            if args[:2] == ["rbd", "export-diff"]:
                Path(args[-1]).write_bytes(("diff for " + args[-2]).encode())
            return subprocess.CompletedProcess(args, 0, "", "")

        with patch.object(worker, "run", side_effect=run):
            worker.backup_task("backup-full", dict(pool="rbd", image="source"), "baseline")
            worker.backup_task("backup-incremental", dict(pool="rbd", image="source", baseBackup="baseline"), "increment")
            items = worker.chain("increment")
        self.assertEqual([item["id"] for item in items], ["baseline", "increment"])
        self.assertEqual(items[-1]["fromSnapshot"], "autoops-baseline")
        self.assertEqual(list(worker.ROOT.glob("*.partial")), [])
        exports = [cmd for cmd in commands if cmd[:2] == ["rbd", "export-diff"]]
        self.assertNotIn("--from-snap", exports[0])
        self.assertEqual(exports[1][2:4], ["--from-snap", "autoops-baseline"])

    def test_subprocess_capture_success_and_nonzero_exit(self):
        result = worker.run([sys.executable, "-c", "print('ok')"], capture=True)
        self.assertEqual(result.stdout.strip(), "ok")
        with self.assertRaisesRegex(RuntimeError, "denied"):
            worker.run([sys.executable, "-c", "import sys; print('denied', file=sys.stderr); sys.exit(13)"], capture=True)

    def test_migration_commit_requires_executed_and_no_watchers(self):
        request = dict(operation="rbd-migration-commit", params=dict(pool="rbd", image="target"))
        for status in ({"watchers": [{"address": "client"}]}, {"migration": {"state": "prepared"}}):
            with self.subTest(status=status), patch.dict(os.environ, TASK_ID="task"), patch.object(worker, "data", return_value=status), patch.object(worker, "run") as run:
                with self.assertRaises((ValueError, RuntimeError)):
                    worker.maintenance(request)
                run.assert_not_called()


if __name__ == "__main__":
    unittest.main()
