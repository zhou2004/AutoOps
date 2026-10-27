"""Executed by a Kubernetes Job using the cluster's Ceph toolbox image.

All subprocess calls use argv, never a shell. Backups use the rbd diff format
for both the full baseline and subsequent increments so snapshot ancestry is
preserved during restore. The repository lock also serializes deletion/restore.
"""
import contextlib
import fcntl
import hashlib
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path("/autoops-backups")
NAME = re.compile(r"^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$")


def run(args, capture=False, check=True):
    print("执行: " + " ".join(args), flush=True)
    # Ceph images based on EL8 may still include Python 3.6 (text= was added in 3.7).
    result = subprocess.run(args, universal_newlines=True, stdout=subprocess.PIPE if capture else None,
                            stderr=subprocess.PIPE if capture else None)
    if check and result.returncode:
        # Commands handling RGW users never print returned JSON (it contains keys).
        raise RuntimeError("命令失败 (%s): %s" % (result.returncode, (result.stderr or "请查看上方日志")[:4000]))
    return result


def data(args):
    return json.loads(run(args, capture=True).stdout)


def image_free(spec):
    if data(["rbd", "status", spec, "--format", "json"]).get("watchers"):
        raise RuntimeError("镜像仍有 watchers，请先停止业务并卸载卷")


def target_absent(spec):
    result = run(["rbd", "info", spec, "--format", "json"], capture=True, check=False)
    if result.returncode == 0:
        raise RuntimeError("目标镜像已存在，拒绝覆盖")
    if result.returncode != 2:
        raise RuntimeError("无法确认目标不存在: " + (result.stderr or ""))


def digest(path):
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(8 * 1024 * 1024), b""):
            value.update(block)
    return value.hexdigest()


def safe_file(backup_id, suffix):
    if not isinstance(backup_id, str) or not NAME.fullmatch(backup_id):
        raise ValueError("备份编号无效")
    path = ROOT / (backup_id + suffix)
    if path.is_symlink():
        raise ValueError("备份路径不能为符号链接")
    return path


def manifest(backup_id):
    with safe_file(backup_id, ".json").open() as stream:
        value = json.load(stream)
    if not isinstance(value, dict) or value.get("version") != 1 or value.get("id") != backup_id:
        raise ValueError("备份清单版本或编号不匹配")
    for key in ("pool", "image", "snapshot", "fsid"):
        if not isinstance(value.get(key), str) or not NAME.fullmatch(value[key]):
            raise ValueError("备份清单资源名无效")
    for key in ("parent", "fromSnapshot"):
        if not isinstance(value.get(key), str) or (value[key] and not NAME.fullmatch(value[key])):
            raise ValueError("备份清单父级关系无效")
    if bool(value["parent"]) != bool(value["fromSnapshot"]):
        raise ValueError("父备份与起始快照必须同时指定")
    for key in ("size", "bytes"):
        if type(value.get(key)) is not int or value[key] <= 0:
            raise ValueError("备份镜像或文件大小无效")
    if not isinstance(value.get("sha256"), str) or not re.fullmatch(r"[0-9a-f]{64}", value["sha256"]):
        raise ValueError("备份校验和格式无效")
    return value


def chain(backup_id, verify=True):
    items, visited = [], set()
    while backup_id:
        if backup_id in visited or len(items) >= 256:
            raise ValueError("备份链存在循环或超过 256 层")
        visited.add(backup_id)
        current = manifest(backup_id)
        path = safe_file(backup_id, ".diff")
        if not path.is_file():
            raise ValueError("备份文件缺失: " + backup_id)
        if path.stat().st_size != current["bytes"]:
            raise ValueError("备份文件大小不匹配: " + backup_id)
        if verify:
            print("校验备份: " + backup_id, flush=True)
            if digest(path) != current.get("sha256"):
                raise ValueError("备份文件校验失败: " + backup_id)
        items.append(current)
        backup_id = current.get("parent", "")
    items.reverse()
    if items[0].get("fromSnapshot"):
        raise ValueError("缺少全量备份基线")
    for parent, child in zip(items, items[1:]):
        if (parent["snapshot"] != child.get("fromSnapshot") or
                any(parent[k] != child[k] for k in ("pool", "image", "fsid"))):
            raise ValueError("增量备份链不连续")
    return items


@contextlib.contextmanager
def repository_lock():
    # Independent jobs, including CronJobs, coordinate using the shared filesystem.
    with (ROOT / ".autoops.lock").open("a") as lock:
        print("等待备份仓库锁", flush=True)
        fcntl.flock(lock, fcntl.LOCK_EX)
        yield


def backup_task(op, p, task_id):
    with repository_lock():
        if op in ("backup-full", "backup-incremental"):
            spec = p["pool"] + "/" + p["image"]
            fsid = run(["ceph", "fsid"], capture=True).stdout.strip()
            snapshot = "autoops-" + task_id
            parent, from_snap = "", ""
            if op == "backup-incremental":
                parent = p["baseBackup"]
                base = chain(parent)[-1]
                if (base["pool"], base["image"], base["fsid"]) != (p["pool"], p["image"], fsid):
                    raise ValueError("父备份不属于当前集群的同一镜像")
                from_snap = base["snapshot"]
                run(["rbd", "info", spec + "@" + from_snap, "--format", "json"], capture=True)
            final = safe_file(task_id, ".diff")
            partial = safe_file(task_id, ".partial")
            meta = safe_file(task_id, ".json")
            if final.exists() or partial.exists() or meta.exists():
                raise ValueError("备份编号已存在，请检查此前任务；不会覆盖或自动重试")
            run(["rbd", "snap", "create", spec + "@" + snapshot])
            info = data(["rbd", "info", spec + "@" + snapshot, "--format", "json"])
            args = ["rbd", "export-diff"]
            if from_snap:
                args += ["--from-snap", from_snap]
            run(args + [spec + "@" + snapshot, str(partial)])
            checksum = digest(partial)
            with partial.open("rb") as stream:
                os.fsync(stream.fileno())
            os.rename(partial, final)
            value = dict(version=1, id=task_id, pool=p["pool"], image=p["image"], fsid=fsid,
                         snapshot=snapshot, fromSnapshot=from_snap, parent=parent,
                         size=info["size"], bytes=final.stat().st_size, sha256=checksum,
                         createdAt=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()))
            temp_meta = safe_file(task_id, ".json.partial")
            with temp_meta.open("x") as stream:
                json.dump(value, stream)
                stream.flush()
                os.fsync(stream.fileno())
            os.rename(temp_meta, meta)
            directory = os.open(str(ROOT), os.O_RDONLY)
            try:
                os.fsync(directory)
            finally:
                os.close(directory)
            print(json.dumps(value, ensure_ascii=False), flush=True)
        elif op in ("backup-restore", "backup-verify"):
            items = chain(p["backupId"])
            if op == "backup-restore":
                target = p["targetPool"] + "/" + p["targetImage"]
                target_absent(target)
                run(["rbd", "create", target, "--size", str((items[0]["size"] + 1048575) // 1048576)])
                for item in items:
                    run(["rbd", "import-diff", str(safe_file(item["id"], ".diff")), target])
                restored = data(["rbd", "info", target, "--format", "json"])
                if restored["size"] != items[-1]["size"]:
                    raise ValueError("恢复后镜像大小与备份清单不一致")
                print("备份链已恢复，请校验文件系统及业务数据后接入应用", flush=True)
            else:
                print("整条备份链校验通过", flush=True)
        elif op == "backup-delete":
            backup_id = p["backupId"]
            manifest(backup_id)
            # Do not delete a baseline still used by any incremental backup.
            for path in ROOT.glob("*.json"):
                item = manifest(path.stem)
                if item.get("parent") == backup_id:
                    raise ValueError("存在依赖此备份的增量: " + item["id"])
            safe_file(backup_id, ".diff").unlink()
            safe_file(backup_id, ".json").unlink()
            print("备份已删除；源快照仍保留", flush=True)
        else:
            raise ValueError("不支持的备份动作")


def maintenance(req):
    op, p = req["operation"], req["params"]
    task_id = os.environ["TASK_ID"]
    if not NAME.fullmatch(task_id):
        raise ValueError("任务编号无效")
    if op.startswith("backup-"):
        return backup_task(op, p, task_id)
    spec = p.get("pool", "") + "/" + p.get("image", "")
    target = p.get("targetPool", "") + "/" + p.get("targetImage", "")
    if op == "rbd-copy":
        target_absent(target)
        run(["rbd", "cp", spec + "@" + p["snapshot"], target])
    elif op == "rbd-deep-copy":
        image_free(spec)
        target_absent(target)
        run(["rbd", "deep", "cp", spec, target])
    elif op.startswith("rbd-migration-"):
        action = op[14:]
        if action not in ("prepare", "execute", "commit", "abort"):
            raise ValueError("不支持的迁移动作")
        if action in ("prepare", "abort", "commit"):
            image_free(spec)
        if action == "prepare":
            target_absent(target)
            run(["rbd", "migration", action, spec, target])
        else:
            status = data(["rbd", "status", spec, "--format", "json"])
            if not status.get("migration"):
                raise ValueError("镜像不处于迁移状态")
            if action == "commit" and status["migration"].get("state") != "executed":
                raise ValueError("只能提交 executed 状态的迁移")
            run(["rbd", "migration", action, spec])
    elif op == "rbd-flatten":
        run(["rbd", "flatten", spec])
    elif op in ("rbd-snap-protect", "rbd-snap-unprotect"):
        run(["rbd", "snap", op[9:], spec + "@" + p["snapshot"]])
    elif op == "rbd-snapshot":
        run(["rbd", "snap", "create", spec + "@autoops-" + task_id])
    elif op.startswith("cephfs-"):
        fs, vol = p["filesystem"], p["subvolume"]
        group = ["--group_name", p["group"]] if p.get("group") else []
        if op == "cephfs-create":
            run(["ceph", "fs", "subvolume", "create", fs, vol, "--size", str(int(p["sizeGiB"]) * 1073741824)] + group)
        elif op == "cephfs-resize":
            run(["ceph", "fs", "subvolume", "resize", fs, vol, str(int(p["sizeGiB"]) * 1073741824), "--no_shrink"] + group)
        elif op == "cephfs-delete":
            run(["ceph", "fs", "subvolume", "rm", fs, vol] + group)
        elif op in ("cephfs-snapshot", "cephfs-snap-delete"):
            run(["ceph", "fs", "subvolume", "snapshot", "create" if op == "cephfs-snapshot" else "rm", fs, vol, p["snapshot"]] + group)
        elif op == "cephfs-clone":
            clone = p["targetSubvolume"]
            target_group = ["--target_group_name", p["group"]] if p.get("group") else []
            run(["ceph", "fs", "subvolume", "snapshot", "clone", fs, vol, p["snapshot"], clone] + group + target_group)
            while True:
                status = data(["ceph", "fs", "clone", "status", fs, clone] + group)
                print(json.dumps(status, ensure_ascii=False), flush=True)
                state = status["status"]["state"]
                if state == "complete":
                    break
                if state not in ("pending", "in-progress"):
                    raise RuntimeError("CephFS 克隆失败或被取消")
                time.sleep(10)
        else:
            raise ValueError("不支持的 CephFS 操作")
    elif op == "osd-scrub":
        run(["ceph", "osd", p["mode"], p["osd"]])
    elif op == "pg-repair":
        run(["ceph", "pg", "repair", p["pg"]])
    elif op == "osd-flag":
        run(["ceph", "osd", p["action"], p["flag"]])
    elif op == "balancer":
        run(["ceph", "balancer", p["action"]])
    elif op == "pool-autoscale":
        run(["ceph", "osd", "pool", "set", p["pool"], "pg_autoscale_mode", p["mode"]])
    elif op in ("rgw-user-enable", "rgw-user-suspend"):
        run(["radosgw-admin", "user", op[9:], "--uid", p["uid"]], capture=True)
        print("RGW 用户状态已更新（响应中的密钥不会写入日志）", flush=True)
    elif op == "rgw-quota":
        run(["radosgw-admin", "quota", "set", "--uid", p["uid"], "--quota-scope", p["scope"], "--max-size", p["sizeGiB"] + "G", "--max-objects", p["maxObjects"]], capture=True)
        run(["radosgw-admin", "quota", "enable", "--uid", p["uid"], "--quota-scope", p["scope"]], capture=True)
    else:
        raise ValueError("不支持的维护动作")


if __name__ == "__main__":
    try:
        maintenance(json.loads(sys.argv[1]))
        print("任务完成", flush=True)
    except Exception as exc:
        print("任务失败: " + str(exc), file=sys.stderr, flush=True)
        sys.exit(1)
