#!/usr/bin/env python3
"""Verify real VPA recommendations and automatic Recreate updates in a new namespace."""
import argparse
from decimal import Decimal
import json
from pathlib import Path
import re
import subprocess
import time


def kubectl(*args, obj=None):
    data = None if obj is None else json.dumps(obj)
    completed = subprocess.run(["kubectl", *args], input=data, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
    return completed.stdout


def get(*args):
    return json.loads(kubectl("get", *args, "-o", "json"))


def cpu_milli(value):
    if value.endswith("m"):
        return Decimal(value[:-1])
    return Decimal(value) * 1000


def pod_evidence(namespace):
    pods = get("pods", "-n", namespace, "-l", "app=autoops-vpa-probe")["items"]
    return [{"name": p["metadata"]["name"], "uid": p["metadata"]["uid"], "node": p["spec"].get("nodeName"), "phase": p["status"].get("phase"), "deleting": "deletionTimestamp" in p["metadata"], "ready": any(c["type"] == "Ready" and c["status"] == "True" for c in p["status"].get("conditions", [])), "resources": p["spec"]["containers"][0]["resources"]} for p in pods]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--namespace", required=True)
    parser.add_argument("--worker", required=True)
    parser.add_argument("--image", required=True, help="Preloaded Alpine image pinned by sha256 digest")
    parser.add_argument("--output", required=True)
    parser.add_argument("--timeout", type=int, default=900)
    parser.add_argument("--cleanup", action="store_true")
    args = parser.parse_args()
    for value in [args.namespace, args.worker]:
        if not re.fullmatch(r"[a-z0-9](?:[-a-z0-9.]{0,61}[a-z0-9])?", value):
            parser.error("namespace and worker must be Kubernetes names")
    if not re.fullmatch(r"[a-z0-9./_-]+@sha256:[a-f0-9]{64}", args.image):
        parser.error("image must use an immutable sha256 digest")
    if not 120 <= args.timeout <= 1800:
        parser.error("timeout must be 120–1800 seconds")
    namespace = args.namespace
    kubectl("create", "namespace", namespace)
    result = {"namespace": namespace, "worker": args.worker, "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "status": "running"}
    try:
        kubectl("label", "namespace", namespace, "app.kubernetes.io/managed-by=autoops-vpa-e2e")
        vpa = {"apiVersion": "autoscaling.k8s.io/v1", "kind": "VerticalPodAutoscaler", "metadata": {"name": "autoops-vpa-probe", "namespace": namespace}, "spec": {"targetRef": {"apiVersion": "apps/v1", "kind": "Deployment", "name": "autoops-vpa-probe"}, "updatePolicy": {"updateMode": "Off", "minReplicas": 2}, "resourcePolicy": {"containerPolicies": [{"containerName": "probe", "controlledResources": ["cpu"], "controlledValues": "RequestsOnly", "minAllowed": {"cpu": "50m"}, "maxAllowed": {"cpu": "250m"}}]}}}
        deployment = {"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"name": "autoops-vpa-probe", "namespace": namespace}, "spec": {"replicas": 2, "selector": {"matchLabels": {"app": "autoops-vpa-probe"}}, "template": {"metadata": {"labels": {"app": "autoops-vpa-probe"}}, "spec": {"nodeSelector": {"kubernetes.io/hostname": args.worker}, "terminationGracePeriodSeconds": 5, "securityContext": {"runAsNonRoot": True, "runAsUser": 65534}, "containers": [{"name": "probe", "image": args.image, "imagePullPolicy": "Never", "command": ["/bin/sh", "-c", "while :; do :; done"], "resources": {"requests": {"cpu": "25m", "memory": "32Mi"}, "limits": {"cpu": "250m", "memory": "128Mi"}}, "securityContext": {"allowPrivilegeEscalation": False, "readOnlyRootFilesystem": True, "capabilities": {"drop": ["ALL"]}}}]}}}}
        budget = {"apiVersion": "policy/v1", "kind": "PodDisruptionBudget", "metadata": {"name": "autoops-vpa-probe", "namespace": namespace}, "spec": {"minAvailable": 1, "selector": {"matchLabels": {"app": "autoops-vpa-probe"}}}}
        kubectl("apply", "-f", "-", obj={"apiVersion": "v1", "kind": "List", "items": [vpa, deployment, budget]})
        kubectl("-n", namespace, "rollout", "status", "deployment/autoops-vpa-probe", "--timeout=180s")
        result["before"] = pod_evidence(namespace)
        if len(result["before"]) != 2 or any(p["resources"]["requests"]["cpu"] != "25m" for p in result["before"]):
            raise RuntimeError("Off mode unexpectedly changed initial CPU requests")
        print("Two initial Pods are ready with CPU request 25m, limit 250m; waiting for real metrics recommendations", flush=True)
        deadline = time.monotonic() + args.timeout
        off_observed_at = None
        result["off_observations"] = []
        initial_uids = {p["uid"] for p in result["before"]}
        while time.monotonic() < deadline:
            observed = get("vpa", "autoops-vpa-probe", "-n", namespace)
            recommendations = observed.get("status", {}).get("recommendation", {}).get("containerRecommendations", [])
            off_pods = pod_evidence(namespace)
            if {p["uid"] for p in off_pods} != initial_uids or any(p["resources"]["requests"]["cpu"] != "25m" or p["deleting"] or not p["ready"] for p in off_pods):
                raise RuntimeError("Off Pods changed or lost readiness before mode switch")
            if recommendations:
                if off_observed_at is None:
                    off_observed_at = time.monotonic()
                held_seconds = int(time.monotonic() - off_observed_at)
                result["off_observations"].append({"at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "held_seconds": held_seconds, "recommendation": observed["status"], "pods": off_pods})
                # Initial recommendations can equal the declared request and be
                # raised only by minAllowed. Require a measured uncapped target
                # and keep Off through at least one default updater interval.
                sampled = cpu_milli(recommendations[0].get("uncappedTarget", {}).get("cpu", "0")) > 25
                if held_seconds >= 90 and sampled:
                    result["off_recommendation"] = observed["status"]
                    result["off_pods"] = off_pods
                    result["off_observation_seconds"] = held_seconds
                    print("Off held unchanged for " + str(held_seconds) + " seconds with a measured recommendation", flush=True)
                    break
                print("Off recommendation observed: " + json.dumps(recommendations), flush=True)
            else:
                print("Waiting for recommendation...", flush=True)
            time.sleep(15)
        else:
            raise TimeoutError("No measured VPA recommendation and 90-second unchanged Off observation before timeout")
        kubectl("-n", namespace, "patch", "vpa", "autoops-vpa-probe", "--type=merge", "-p", '{"spec":{"updatePolicy":{"updateMode":"Recreate","minReplicas":2}}}')
        result["recreate_started_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        while time.monotonic() < deadline:
            pods = pod_evidence(namespace)
            current = [p for p in pods if not p["deleting"]]
            if len(current) == 2 and all(p["uid"] not in initial_uids and p["ready"] and cpu_milli(p["resources"]["requests"]["cpu"]) > 25 for p in current):
                result["after"] = current
                result["final_vpa_status"] = get("vpa", "autoops-vpa-probe", "-n", namespace).get("status")
                result["deployment_template_resources"] = get("deployment", "autoops-vpa-probe", "-n", namespace)["spec"]["template"]["spec"]["containers"][0]["resources"]
                result["events"] = get("events", "-n", namespace)["items"]
                if result["deployment_template_resources"]["requests"]["cpu"] != "25m":
                    raise RuntimeError("Deployment template changed; cannot attribute result only to VPA admission")
                result["status"] = "passed"
                print("PASS: both Pods were automatically recreated with increased CPU requests; Deployment template remains 25m", flush=True)
                break
            print("Waiting for automatic Recreate: " + json.dumps([{"name": p["name"], "cpu": p["resources"]["requests"]["cpu"], "ready": p["ready"], "deleting": p["deleting"]} for p in pods]), flush=True)
            time.sleep(15)
        else:
            raise TimeoutError("VPA did not automatically replace both Pods before timeout")
    except Exception as exc:
        result["status"] = "failed"
        result["error"] = str(exc)
        raise
    finally:
        result["finished_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        try:
            if args.cleanup:
                kubectl("delete", "namespace", namespace, "--wait=true", "--timeout=120s")
                result["namespace_deleted"] = True
                print("Deleted only the dedicated test namespace: " + namespace, flush=True)
        finally:
            Path(args.output).write_text(json.dumps(result, indent=2) + "\n")


if __name__ == "__main__":
    main()
