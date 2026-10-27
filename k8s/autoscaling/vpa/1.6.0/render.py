#!/usr/bin/env python3
"""Render pinned upstream VPA manifests with explicit worker placement."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

import yaml

VERSION = "1.6.0"
UPSTREAM = Path(__file__).resolve().parent / "upstream"


def render(workers, compact=False, stage="all", admission_worker=None):
    worker_list = workers.split(",")
    if not worker_list or any(not re.fullmatch(r"[a-z0-9](?:[-a-z0-9.]{0,251}[a-z0-9])?", item) for item in worker_list):
        raise ValueError("--workers must be comma-separated Kubernetes node hostnames")
    if admission_worker is not None and admission_worker not in worker_list:
        raise ValueError("--admission-worker must be one of --workers")
    resources = []
    for line in (UPSTREAM / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split()
        raw = (UPSTREAM / name).read_bytes()
        if hashlib.sha256(raw).hexdigest() != expected:
            raise ValueError(f"Pinned upstream checksum mismatch: {name}")
        for resource in yaml.safe_load_all(raw):
            if resource is None:
                continue
            is_controller = resource["kind"] == "Deployment"
            if stage == "prerequisites" and is_controller:
                continue
            if stage == "controllers" and not is_controller:
                continue
            meta = resource["metadata"]
            meta.setdefault("labels", {}).update({"app.kubernetes.io/managed-by": "autoops-vpa", "app.kubernetes.io/version": VERSION})
            # Avoid taking over generic upstream names such as system:metrics-reader.
            if resource["kind"] in {"ClusterRole", "ClusterRoleBinding", "Role", "RoleBinding"}:
                meta["name"] = "autoops:vpa:" + meta["name"].removeprefix("system:")
                if "roleRef" in resource:
                    resource["roleRef"]["name"] = "autoops:vpa:" + resource["roleRef"]["name"].removeprefix("system:")
            if is_controller:
                spec = resource["spec"]["template"]["spec"]
                placement = [admission_worker] if admission_worker and meta["name"] == "vpa-admission-controller" else worker_list
                spec["affinity"] = {"nodeAffinity": {"requiredDuringSchedulingIgnoredDuringExecution": {"nodeSelectorTerms": [{"matchExpressions": [{"key": "kubernetes.io/hostname", "operator": "In", "values": placement}]}]}}}
                resource["spec"]["template"]["metadata"].setdefault("labels", {})["app.kubernetes.io/managed-by"] = "autoops-vpa"
                if compact:
                    recommender = meta["name"] == "vpa-recommender"
                    spec["containers"][0]["resources"] = {"requests": {"cpu": "50m", "memory": "128Mi" if recommender else "64Mi"}, "limits": {"cpu": "300m", "memory": "384Mi" if recommender else "192Mi"}}
            resources.append(resource)
    return {"apiVersion": "v1", "kind": "List", "items": resources}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", required=True)
    parser.add_argument("--admission-worker", help="Optional cached-image worker for the admission controller")
    parser.add_argument("--compact", action="store_true", help="Small-cluster resource profile; monitor memory before production use")
    parser.add_argument("--stage", choices=["all", "prerequisites", "controllers"], default="all")
    args = parser.parse_args()
    try:
        json.dump(render(args.workers, args.compact, args.stage, args.admission_worker), sys.stdout, indent=2)
        print()
    except (OSError, ValueError) as exc:
        parser.error(str(exc))


if __name__ == "__main__":
    main()
