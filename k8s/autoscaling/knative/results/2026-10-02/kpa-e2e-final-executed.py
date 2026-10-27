#!/usr/bin/env python3
"""Bounded live KPA test using a new namespace; cleans up only its own namespace."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import subprocess
import threading
import time
import urllib.request
import uuid


IMAGE = "docker.io/library/python@sha256:4c47124a8391cb7a9f571164147d154777cf012a4ece5f86097130d7a4478111"
APP = """import http.server, time, signal, sys
signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != '/health': time.sleep(0.25)
        body = b'autoops-kpa-ok\\n'
        self.send_response(200); self.send_header('Content-Length', str(len(body)))
        self.end_headers(); self.wfile.write(body)
    def log_message(self, *args): pass
http.server.ThreadingHTTPServer(('0.0.0.0', 8080), Handler).serve_forever()
"""


def kubectl(*args, obj=None, timeout=45):
    result = subprocess.run(["kubectl", *args], input=json.dumps(obj) if obj is not None else None,
                            text=True, capture_output=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return result.stdout


def emit(event, **data):
    print(json.dumps({"time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                      "event": event, **data}), flush=True)


def run(namespace, duration, metric):
    token = str(uuid.uuid4())
    existing = kubectl("get", "namespace", namespace, "--ignore-not-found", "-o", "name").strip()
    if existing:
        raise RuntimeError(f"Refusing to reuse existing namespace {namespace}")
    created = json.loads(kubectl("create", "-f", "-", "-o", "json", obj={"apiVersion": "v1", "kind": "Namespace", "metadata": {
        "name": namespace, "labels": {"autoops.io/test": "kpa-e2e"}, "annotations": {"autoops.io/test-run": token}}}))
    namespace_uid = created["metadata"]["uid"]
    service_uid = None
    try:
        annotations = {
            "autoscaling.knative.dev/class": "kpa.autoscaling.knative.dev",
            "autoscaling.knative.dev/metric": metric,
            "autoscaling.knative.dev/target": "1" if metric == "concurrency" else "5",
            "autoscaling.knative.dev/min-scale": "0",
            "autoscaling.knative.dev/max-scale": "3",
            "autoscaling.knative.dev/window": "30s",
            "queue.sidecar.serving.knative.dev/cpu-resource-request": "25m",
            "queue.sidecar.serving.knative.dev/cpu-resource-limit": "200m",
            "queue.sidecar.serving.knative.dev/memory-resource-request": "50Mi",
            "queue.sidecar.serving.knative.dev/memory-resource-limit": "96Mi",
        }
        service = {"apiVersion": "serving.knative.dev/v1", "kind": "Service", "metadata": {
            "name": "kpa-check", "namespace": namespace}, "spec": {"template": {
                "metadata": {"annotations": annotations}, "spec": {
                    "nodeSelector": {"kubernetes.io/role": "node"}, "containerConcurrency": 4,
                    "containers": [{"name": "http", "image": IMAGE, "imagePullPolicy": "IfNotPresent",
                                    "command": ["python3", "-u", "-c", APP], "ports": [{"containerPort": 8080}],
                                    "resources": {"requests": {"cpu": "25m", "memory": "32Mi"},
                                                  "limits": {"cpu": "200m", "memory": "96Mi"}},
                                    "readinessProbe": {"httpGet": {"path": "/health", "port": 8080}}}]}}}}
        created_service = json.loads(kubectl("create", "-f", "-", "-o", "json", obj=service))
        service_uid = created_service["metadata"]["uid"]
        # Poll with bounded individual calls so the outer operator can monitor progress.
        deadline = time.monotonic() + 240
        while True:
            current = json.loads(kubectl("-n", namespace, "get", "ksvc", "kpa-check", "-o", "json"))
            ready = any(c["type"] == "Ready" and c["status"] == "True" for c in current.get("status", {}).get("conditions", []))
            if ready:
                break
            if time.monotonic() > deadline:
                raise RuntimeError("Knative Service did not become Ready: " + json.dumps(current.get("status", {})))
            time.sleep(3)
        host = current["status"]["url"].split("://", 1)[1].split("/", 1)[0]
        # The fresh installation keeps the default .svc.cluster.local domain;
        # those routes are deliberately served by Kourier's internal listener.
        gateway = kubectl("-n", "kourier-system", "get", "service", "kourier-internal", "-o", "jsonpath={.spec.clusterIP}").strip()
        url = f"http://{gateway}/"

        def replicas():
            deployments = json.loads(kubectl("-n", namespace, "get", "deployments", "-l", "serving.knative.dev/service=kpa-check", "-o", "json"))["items"]
            pods = json.loads(kubectl("-n", namespace, "get", "pods", "-l", "serving.knative.dev/service=kpa-check", "-o", "json"))["items"]
            return {"desired": sum(d["spec"].get("replicas", 0) for d in deployments),
                    "ready": sum(d.get("status", {}).get("readyReplicas", 0) for d in deployments), "pods": len(pods)}

        def wait_zero(phase):
            deadline = time.monotonic() + 240
            while True:
                count = replicas()
                emit("replicas", phase=phase, **count)
                if count == {"desired": 0, "ready": 0, "pods": 0}:
                    return
                if time.monotonic() > deadline:
                    raise RuntimeError(f"Did not scale to zero during {phase}")
                time.sleep(5)

        # Explicit empty ProxyHandler keeps private cluster traffic off any HTTP proxy.
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

        def request():
            started = time.monotonic()
            with opener.open(urllib.request.Request(url, headers={"Host": host}), timeout=60) as response:
                body = response.read()
                if response.status != 200 or body != b"autoops-kpa-ok\n":
                    raise RuntimeError("HTTP response did not match the test application")
            return round(time.monotonic() - started, 3)

        wait_zero("initial-idle")
        emit("cold-start", seconds=request())
        stats = {"ok": 0, "errors": 0}
        lock = threading.Lock()
        end = time.monotonic() + duration

        def traffic():
            while time.monotonic() < end:
                try:
                    request()
                    key = "ok"
                except Exception:
                    key = "errors"
                with lock:
                    stats[key] += 1

        peak = 0
        with ThreadPoolExecutor(max_workers=8) as pool:
            workers = [pool.submit(traffic) for _ in range(8)]
            while time.monotonic() < end:
                count = replicas()
                peak = max(peak, count["ready"])
                emit("replicas", phase="traffic", **count)
                if count["desired"] > 3 or count["ready"] > 3:
                    raise RuntimeError("KPA exceeded max-scale=3")
                time.sleep(5)
            for worker in workers:
                worker.result()
        emit("traffic-completed", peak_ready=peak, **stats)
        if peak < 2 or stats["ok"] == 0 or stats["errors"]:
            raise RuntimeError("Expected >=2 ready replicas and successful requests without errors")
        wait_zero("after-traffic")
        emit("cold-restart", seconds=request())
        emit("functional-passed", namespace=namespace, metric=metric, peak_ready=peak, **stats)
    finally:
        raw = kubectl("get", "namespace", namespace, "--ignore-not-found", "-o", "json")
        if raw.strip():
            current = json.loads(raw)
            if current["metadata"]["uid"] != namespace_uid or current["metadata"].get("annotations", {}).get("autoops.io/test-run") != token:
                raise RuntimeError("Namespace ownership changed; refusing cleanup")
            try:
                raw_service = kubectl("-n", namespace, "get", "services.serving.knative.dev/kpa-check", "--ignore-not-found", "-o", "json")
                if service_uid and raw_service.strip():
                    kubectl("delete", "--raw", f"/apis/serving.knative.dev/v1/namespaces/{namespace}/services/kpa-check", "-f", "-",
                            obj={"apiVersion": "v1", "kind": "DeleteOptions", "propagationPolicy": "Foreground", "preconditions": {"uid": service_uid}})
                    kubectl("-n", namespace, "wait", "--for=delete", "services.serving.knative.dev/kpa-check", "--timeout=120s", timeout=130)
            finally:
                # UID preconditions are enforced by the API server, closing the
                # race between the ownership read and a namespace replacement.
                kubectl("delete", "--raw", f"/api/v1/namespaces/{namespace}", "-f", "-",
                        obj={"apiVersion": "v1", "kind": "DeleteOptions", "preconditions": {"uid": namespace_uid}})
                emit("cleanup-requested", namespace=namespace)
                kubectl("wait", "--for=delete", f"namespace/{namespace}", "--timeout=300s", timeout=310)
        emit("cleanup-confirmed", namespace=namespace)
    emit("passed", namespace=namespace, metric=metric, peak_ready=peak, **stats)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--namespace", default="autoops-kpa-e2e-20261002")
    parser.add_argument("--duration", type=int, default=75)
    parser.add_argument("--metric", choices=["concurrency", "rps"], default="concurrency")
    args = parser.parse_args()
    if not args.namespace.startswith("autoops-kpa-e2e-") or not 30 <= args.duration <= 120:
        parser.error("Use an autoops-kpa-e2e-* namespace and 30–120 seconds of traffic")
    run(args.namespace, args.duration, args.metric)
