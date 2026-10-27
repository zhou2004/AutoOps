#!/usr/bin/env python3
"""Download and render the reviewed Knative 1.23.0 lab installation. No kubectl."""
import argparse
import hashlib
import json
from pathlib import Path
import urllib.request

import yaml


VERSION = "1.23.0"
PROFILE = "bounded-lab-20261002"
SOURCES = {
    "serving-crds.yaml": ("knative/serving", "b172ff4901ed50f8e4e09ff8616e54d22e264df7086ce8cb74f513a04812fe74"),
    "serving-core.yaml": ("knative/serving", "be3f16c9c0ac9276cc173ef04871aaeac78537f9edb116310caa02f016e9cbc2"),
    "kourier.yaml": ("knative-extensions/net-kourier", "cded0c3c1d7669b1aa9f7484234b454ff3940a2b54a27d5ec4825c2d4003d01d"),
}
MEMORY = {
    "activator": ("64Mi", "256Mi"),
    "autoscaler": ("96Mi", "256Mi"),
    "controller": ("96Mi", "384Mi"),
    "webhook": ("96Mi", "256Mi"),
    "net-kourier-controller": ("64Mi", "256Mi"),
    "3scale-kourier-gateway": ("128Mi", "256Mi"),
}


def prepare(output, envoy_image, cache=None):
    if not envoy_image.startswith("docker.io/envoyproxy/envoy@sha256:") or len(envoy_image.rsplit(":", 1)[1]) != 64:
        raise ValueError("Resolve the official envoy:v1.37-latest image to its full digest first")
    output.mkdir(parents=True, exist_ok=True)
    images = {envoy_image}
    inventory = []
    for filename, (repo, expected) in SOURCES.items():
        url = f"https://github.com/{repo}/releases/download/knative-v{VERSION}/{filename}"
        raw = (cache / filename).read_bytes() if cache else urllib.request.urlopen(url, timeout=60).read()
        if hashlib.sha256(raw).hexdigest() != expected:
            raise ValueError(f"Official manifest checksum mismatch: {filename}")
        docs = [item for item in yaml.safe_load_all(raw) if item]
        for item in docs:
            meta = item["metadata"]
            meta.setdefault("labels", {})["autoops.io/install-profile"] = PROFILE
            name = meta["name"]
            inventory.append({"kind": item["kind"], "namespace": meta.get("namespace"), "name": name})
            if item["kind"] == "Deployment":
                item["spec"]["replicas"] = 1
                pod = item["spec"]["template"]["spec"]
                pod["nodeSelector"] = {"kubernetes.io/role": "node"}
                request_memory, limit_memory = MEMORY[name]
                for container in pod["containers"]:
                    container["resources"] = {
                        "requests": {"cpu": "100m", "memory": request_memory},
                        "limits": {"cpu": "500m", "memory": limit_memory},
                    }
                    container["imagePullPolicy"] = "IfNotPresent"
                    if name == "3scale-kourier-gateway":
                        container["image"] = envoy_image
                    images.add(container["image"])
            if item["kind"] == "HorizontalPodAutoscaler":
                item["spec"]["minReplicas"] = item["spec"]["maxReplicas"] = 1
            if item["kind"] == "Service" and name == "kourier":
                item["spec"]["type"] = "ClusterIP"
            if item["kind"] == "ConfigMap":
                data = item.setdefault("data", {})
                if name == "config-network":
                    data["ingress-class"] = "kourier.ingress.networking.knative.dev"
                if name == "config-features":
                    data["kubernetes.podspec-nodeselector"] = "enabled"
                if name == "config-deployment":
                    images.add(data["queue-sidecar-image"])
        (output / filename).write_text(yaml.safe_dump_all(docs, sort_keys=False), encoding="utf-8")
    (output / "images.txt").write_text("\n".join(sorted(images)) + "\n", encoding="utf-8")
    (output / "review.json").write_text(json.dumps({
        "version": VERSION, "minimum_kubernetes": "1.34.0", "profile": PROFILE,
        "upstream_sha256": {key: value[1] for key, value in SOURCES.items()},
        "images": sorted(images), "objects": inventory,
    }, indent=2) + "\n", encoding="utf-8")
    print(f"Prepared {output}; no cluster changes performed")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--envoy-image", required=True)
    parser.add_argument("--cache", type=Path, help="Directory containing original verified upstream manifests")
    args = parser.parse_args()
    prepare(args.output, args.envoy_image, args.cache)
