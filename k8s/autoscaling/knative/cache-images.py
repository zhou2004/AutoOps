#!/usr/bin/env python3
"""Make verified amd64 OCI archives for containerd without changing a cluster."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile


def archive_image(crane, image, output):
    if not re.fullmatch(r"[a-zA-Z0-9._:/-]+@sha256:[0-9a-f]{64}", image):
        raise ValueError("Only full image@sha256 references are accepted")
    name, digest = image.split("@sha256:")
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="knative-image-", dir=output) as temp:
        layout = Path(temp) / "layout"
        subprocess.run([crane, "pull", "--platform=linux/amd64", "--format=oci", image, str(layout)],
                       check=True, timeout=600)
        # Preserve the original multiarch index so the cached reference remains
        # the published digest; only the selected amd64 layers are downloaded.
        raw = subprocess.run([crane, "manifest", image], capture_output=True, check=True, timeout=60).stdout
        if hashlib.sha256(raw).hexdigest() != digest:
            raw = raw.rstrip(b"\n")
        if hashlib.sha256(raw).hexdigest() != digest:
            raise ValueError("Registry manifest does not match its pinned digest")
        blobs = layout / "blobs" / "sha256"
        (blobs / digest).write_bytes(raw)
        for blob in blobs.iterdir():
            if hashlib.sha256(blob.read_bytes()).hexdigest() != blob.name:
                raise ValueError(f"Incomplete or corrupt OCI blob: {blob.name}")
        index = {"schemaVersion": 2, "mediaType": "application/vnd.oci.image.index.v1+json", "manifests": [{
            "mediaType": json.loads(raw)["mediaType"], "size": len(raw), "digest": "sha256:" + digest,
            "annotations": {"io.containerd.image.name": image, "org.opencontainers.image.ref.name": image}}]}
        (layout / "index.json").write_text(json.dumps(index), encoding="utf-8")
        filename = name.replace("/", "_").replace(":", "_") + "_" + digest[:12] + ".tar"
        staging = Path(temp) / filename
        with tarfile.open(staging, "w") as archive:
            for path in layout.iterdir():
                archive.add(path, arcname=path.name)
        destination = output / filename
        staging.replace(destination)
        print(json.dumps({"image": image, "archive": str(destination), "bytes": destination.stat().st_size}), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--crane", default="crane")
    parser.add_argument("--images-file", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    for image in args.images_file.read_text(encoding="utf-8").splitlines():
        if image.strip():
            archive_image(args.crane, image.strip(), args.output)
