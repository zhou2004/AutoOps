#!/usr/bin/env python3
"""Bounded HPA scale-out/scale-in test against the selected kubectl context.

Creates a new namespace, never reuses one, and removes it in finally. Run only
against an explicitly approved cluster. No third-party Python modules needed.
"""
import argparse
import datetime
import json
import pathlib
import re
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--namespace", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--node", required=True, help="Approved worker hostname")
    parser.add_argument("--image-pull-policy", choices=["Never", "IfNotPresent"], default="IfNotPresent")
    parser.add_argument("--output", required=True, help="Evidence JSON path outside the namespace")
    args = parser.parse_args()
    if not re.fullmatch(r"autoops-hpa-e2e-[a-z0-9-]{1,35}", args.namespace):
        parser.error("namespace must be a new autoops-hpa-e2e-* namespace")
    report = {"namespace": args.namespace, "node": args.node, "image": args.image, "observations": [], "passed": False}
    created_uid = None
    # Absolute deadline survives container restarts and stops load even if this
    # process loses access to the API before it can switch the ConfigMap off.
    load_expires_at = int(time.time()) + 1800
    report["loadExpiresAtUnix"] = load_expires_at

    def kubectl(*words, obj=None):
        result = subprocess.run(
            ["kubectl", "--request-timeout=25s", *words],
            input=None if obj is None else json.dumps(obj), text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=35,
        )
        if result.returncode:
            raise RuntimeError("kubectl " + " ".join(words) + ": " + result.stderr.strip())
        return result.stdout

    def apply(obj):
        kubectl("apply", "-f", "-", obj=obj)

    def snapshot(phase):
        hpa = json.loads(kubectl("-n", args.namespace, "get", "hpa", "cpu", "-o", "json"))
        deploy = json.loads(kubectl("-n", args.namespace, "get", "deployment", "cpu", "-o", "json"))
        status = hpa.get("status", {})
        obs = {"at": datetime.datetime.now(datetime.timezone.utc).isoformat(), "phase": phase,
               "desired": status.get("desiredReplicas"), "current": status.get("currentReplicas"),
               "deploymentReplicas": deploy["spec"]["replicas"], "ready": deploy.get("status", {}).get("readyReplicas", 0),
               "metrics": status.get("currentMetrics", []), "conditions": status.get("conditions", [])}
        report["observations"].append(obs)
        print(json.dumps(obs, ensure_ascii=False), flush=True)
        return obs

    def wait_for(phase, check, timeout):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            obs = snapshot(phase)
            if check(obs):
                return obs
            time.sleep(10)
        raise RuntimeError(phase + " did not reach the required state before the deadline")

    def metric_active(obs):
        return bool(obs["metrics"]) and any(x["type"] == "ScalingActive" and x["status"] == "True" for x in obs["conditions"])

    try:
        report["serverVersion"] = json.loads(kubectl("version", "-o", "json"))["serverVersion"]["gitVersion"]
        # `create` refuses an existing namespace; never adopt or clean up somebody else's resources.
        created_namespace = json.loads(kubectl("create", "-f", "-", "-o", "json", obj={
            "apiVersion": "v1", "kind": "Namespace", "metadata": {"name": args.namespace,
                "labels": {"app.kubernetes.io/managed-by": "autoops-live-test"}}}))
        created_uid = created_namespace["metadata"]["uid"]
        report["namespaceUID"] = created_uid
        ns = {"namespace": args.namespace}
        apply({"apiVersion": "v1", "kind": "ResourceQuota", "metadata": {**ns, "name": "test-budget"},
               "spec": {"hard": {"pods": "4", "requests.cpu": "200m", "limits.cpu": "400m", "requests.memory": "128Mi", "limits.memory": "256Mi"}}})
        apply({"apiVersion": "v1", "kind": "ConfigMap", "metadata": {**ns, "name": "load-control"}, "data": {"load": "off"}})
        apply({"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {**ns, "name": "cpu"}, "spec": {
            "replicas": 1, "selector": {"matchLabels": {"app": "hpa-cpu-test"}},
            "template": {"metadata": {"labels": {"app": "hpa-cpu-test"}}, "spec": {
                "nodeSelector": {"kubernetes.io/hostname": args.node}, "automountServiceAccountToken": False,
                "terminationGracePeriodSeconds": 2,
                "containers": [{"name": "cpu", "image": args.image, "imagePullPolicy": args.image_pull_policy,
                    "command": ["/bin/sh", "-c", 'while true; do if [ "$(date +%s)" -lt ' + str(load_expires_at) + ' ] && [ "$(cat /control/load)" = on ]; then timeout 2 /bin/sh -c "while :; do :; done"; else sleep 2; fi; done'],
                    "resources": {"requests": {"cpu": "20m", "memory": "16Mi"}, "limits": {"cpu": "100m", "memory": "64Mi"}},
                    "securityContext": {"allowPrivilegeEscalation": False, "capabilities": {"drop": ["ALL"]}},
                    "volumeMounts": [{"name": "control", "mountPath": "/control", "readOnly": True}]}],
                "volumes": [{"name": "control", "configMap": {"name": "load-control"}}]}}}})
        apply({"apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscaler", "metadata": {**ns, "name": "cpu"}, "spec": {
            "scaleTargetRef": {"apiVersion": "apps/v1", "kind": "Deployment", "name": "cpu"}, "minReplicas": 1, "maxReplicas": 3,
            "metrics": [{"type": "Resource", "resource": {"name": "cpu", "target": {"type": "Utilization", "averageUtilization": 50}}}],
            "behavior": {"scaleUp": {"stabilizationWindowSeconds": 0}, "scaleDown": {"stabilizationWindowSeconds": 30,
                "policies": [{"type": "Pods", "value": 2, "periodSeconds": 15}]}}}})
        report["idle"] = wait_for("idle", lambda x: metric_active(x) and x["ready"] == 1 and x["desired"] == 1, 240)
        kubectl("-n", args.namespace, "patch", "configmap", "load-control", "--type=merge", "-p", '{"data":{"load":"on"}}')
        report["scaleOut"] = wait_for("load-on", lambda x: metric_active(x) and x["desired"] == 3 and x["ready"] == 3, 480)
        report["podImages"] = [{"name": p["metadata"]["name"], "node": p["spec"].get("nodeName"), "images": p.get("status", {}).get("containerStatuses", [])}
                               for p in json.loads(kubectl("-n", args.namespace, "get", "pods", "-o", "json"))["items"]]
        kubectl("-n", args.namespace, "patch", "configmap", "load-control", "--type=merge", "-p", '{"data":{"load":"off"}}')
        report["scaleIn"] = wait_for("load-off", lambda x: metric_active(x) and x["desired"] == 1 and x["deploymentReplicas"] == 1 and x["ready"] == 1, 420)
        report["events"] = json.loads(kubectl("-n", args.namespace, "get", "events", "-o", "json"))
        report["functionalPassed"] = True
    except Exception as exc:
        report["error"] = str(exc)
    finally:
        if created_uid:
            try:
                # Atomic UID precondition prevents deleting a replacement with
                # the same name. Do not alter namespace finalizers.
                kubectl("delete", "--raw", "/api/v1/namespaces/" + args.namespace, "-f", "-", obj={
                    "apiVersion": "v1", "kind": "DeleteOptions", "preconditions": {"uid": created_uid}})
                report["cleanupRequested"] = True
                cleanup_deadline = time.monotonic() + 180
                while time.monotonic() < cleanup_deadline:
                    raw = kubectl("get", "namespace", args.namespace, "--ignore-not-found", "-o", "json")
                    if not raw.strip():
                        report["namespaceDeleted"] = True
                        break
                    if json.loads(raw)["metadata"]["uid"] != created_uid:
                        raise RuntimeError("namespace UID changed; refusing to clean up its replacement")
                    time.sleep(5)
                else:
                    raise RuntimeError("namespace deletion did not finish within 180 seconds")
            except Exception as exc:
                report["cleanupError"] = str(exc)
        report["passed"] = bool(report.get("functionalPassed") and report.get("namespaceDeleted") and not report.get("cleanupError"))
        pathlib.Path(args.output).write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
        print(json.dumps({k: report.get(k) for k in ["passed", "functionalPassed", "error", "namespaceDeleted", "cleanupError"]}), flush=True)
    if not report["passed"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
