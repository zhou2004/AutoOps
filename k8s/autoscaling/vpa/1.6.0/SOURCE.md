# VPA 1.6.0 source and local changes

- Upstream: https://github.com/kubernetes/autoscaler
- Release tag: `vertical-pod-autoscaler-1.6.0`
- Commit: `9196162ba12e637ce4482107e1c17bcb80599050`
- Upstream files: `vertical-pod-autoscaler/deploy/`, copied without modification into `upstream/`.
- File integrity: `upstream/SHA256SUMS`; `render.py` checks these before rendering.
- License: upstream Apache License 2.0, included in `LICENSE`.
- Container images: `registry.k8s.io/autoscaling/vpa-{recommender,updater,admission-controller}:1.6.0`.

`render.py` applies explicit worker affinity, AutoOps ownership labels, and a separate RBAC name prefix. `--compact` changes only controller resource requests/limits for a small test cluster. `--admission-worker` can select a worker with a verified cached image. The CRD and VPA controller behavior remain upstream 1.6.0.

`install.sh` validates Kubernetes 1.34–1.36, metrics, worker readiness and source checksums. It preserves existing TLS material and refuses to take over controllers owned elsewhere. A new installation generates a one-year private CA and server certificate locally; private files are mode 0600 and removed when the script exits. The resulting Secret stays in `kube-system`. Certificate rotation is an explicit maintenance task; do it before expiry.

`run-e2e.py` creates a fresh isolated namespace, observes Off recommendations for at least 90 seconds and requires a measured uncapped CPU target, switches to Recreate, and verifies both Pod UIDs and CPU requests changed while the Deployment template stayed unchanged. It does not manually restart or delete Pods to obtain the result. `--cleanup` deletes only the namespace created by that invocation.

See [the installation and validation guide](../../../../docs/2026-10-02-vpa-installation-and-validation.md).
