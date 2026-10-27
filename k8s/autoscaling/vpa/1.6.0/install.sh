#!/usr/bin/env bash
# Install only the pinned VPA 1.6.0 manifests in this directory.
set -euo pipefail
umask 077
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
action=${1:-check}
shift || true
case "$action" in check|install) ;; *) echo 'Usage: install.sh check|install --workers worker-01,worker-02 [--compact]' >&2; exit 2 ;; esac
for dependency in kubectl python3 openssl; do command -v "$dependency" >/dev/null || { echo "Missing $dependency" >&2; exit 1; }; done
python3 -c 'import yaml' || { echo 'Python PyYAML is required.' >&2; exit 1; }
work_dir=$(mktemp -d)
trap 'rm -rf -- "$work_dir"' EXIT
python3 "$script_dir/render.py" "$@" --stage prerequisites > "$work_dir/prerequisites.json"
python3 "$script_dir/render.py" "$@" --stage controllers > "$work_dir/controllers.json"
kubectl version -o json > "$work_dir/version.json"
python3 - "$work_dir/version.json" <<'PY'
import json, re, sys
version=json.load(open(sys.argv[1]))['serverVersion']
minor=int(re.match(r'\d+', version['minor']).group())
if version['major'] != '1' or not 34 <= minor <= 36:
    raise SystemExit('This maintained VPA 1.6 recipe supports Kubernetes 1.34–1.36; select a compatible pinned recipe.')
print('Target Kubernetes:', version['gitVersion'])
PY
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes > "$work_dir/metrics.json"
python3 - "$work_dir/metrics.json" <<'PY'
import json,sys
if not json.load(open(sys.argv[1])).get('items'): raise SystemExit('Metrics API has no node samples')
PY
kubectl get nodes -o json > "$work_dir/nodes.json"
python3 - "$work_dir/nodes.json" "$work_dir/controllers.json" <<'PY'
import json,sys
nodes=json.load(open(sys.argv[1]))['items']; manifests=json.load(open(sys.argv[2]))['items']
requested=manifests[0]['spec']['template']['spec']['affinity']['nodeAffinity']['requiredDuringSchedulingIgnoredDuringExecution']['nodeSelectorTerms'][0]['matchExpressions'][0]['values']
for name in requested:
    matches=[n for n in nodes if n['metadata']['labels'].get('kubernetes.io/hostname')==name]
    if len(matches)!=1: raise SystemExit('Unknown worker hostname: '+name)
    node=matches[0]; labels=node['metadata']['labels']
    if 'node-role.kubernetes.io/control-plane' in labels or 'node-role.kubernetes.io/master' in labels: raise SystemExit('Refusing control-plane placement: '+name)
    if not any(c['type']=='Ready' and c['status']=='True' for c in node['status']['conditions']): raise SystemExit('Worker not Ready: '+name)
print('VPA workers:', ', '.join(requested))
PY
for deployment in vpa-recommender vpa-updater vpa-admission-controller; do
  manager=$(kubectl -n kube-system get deployment "$deployment" --ignore-not-found -o 'jsonpath={.metadata.labels.app\.kubernetes\.io/managed-by}')
  if [[ -n "$manager" && "$manager" != autoops-vpa ]]; then echo "Existing $deployment is managed elsewhere; refusing takeover." >&2; exit 1; fi
  if kubectl -n kube-system get deployment "$deployment" >/dev/null 2>&1 && [[ "$manager" != autoops-vpa ]]; then echo "Existing unlabelled $deployment; refusing takeover." >&2; exit 1; fi
done
if [[ "$action" == check ]]; then
  echo 'Source checksums, Kubernetes compatibility, metrics, and worker placement checks passed. No resources changed.'
  exit 0
fi
# Server-side apply avoids the annotation size limit for large CRDs.
kubectl apply --server-side --field-manager=autoops-vpa -f "$work_dir/prerequisites.json"
kubectl wait --for=condition=Established --timeout=90s crd/verticalpodautoscalers.autoscaling.k8s.io crd/verticalpodautoscalercheckpoints.autoscaling.k8s.io
if ! kubectl -n kube-system get secret vpa-tls-certs >/dev/null 2>&1; then
  cat > "$work_dir/server.conf" <<'CONFIG'
[req]
req_extensions = v3_req
distinguished_name = distinguished_name
[distinguished_name]
[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:vpa-webhook.kube-system.svc
CONFIG
  openssl genrsa -out "$work_dir/caKey.pem" 2048 2>/dev/null
  openssl req -x509 -new -nodes -key "$work_dir/caKey.pem" -days 365 -out "$work_dir/caCert.pem" -subj '/CN=autoops-vpa-ca' -addext 'basicConstraints=critical,CA:TRUE' 2>/dev/null
  openssl genrsa -out "$work_dir/serverKey.pem" 2048 2>/dev/null
  openssl req -new -key "$work_dir/serverKey.pem" -out "$work_dir/server.csr" -subj '/CN=vpa-webhook.kube-system.svc' -config "$work_dir/server.conf" 2>/dev/null
  openssl x509 -req -in "$work_dir/server.csr" -CA "$work_dir/caCert.pem" -CAkey "$work_dir/caKey.pem" -CAcreateserial -out "$work_dir/serverCert.pem" -days 365 -extensions v3_req -extfile "$work_dir/server.conf" 2>/dev/null
  kubectl -n kube-system create secret generic vpa-tls-certs --from-file="$work_dir/caKey.pem" --from-file="$work_dir/caCert.pem" --from-file="$work_dir/serverKey.pem" --from-file="$work_dir/serverCert.pem"
  kubectl -n kube-system label secret vpa-tls-certs app.kubernetes.io/managed-by=autoops-vpa
fi
kubectl apply --server-side --field-manager=autoops-vpa -f "$work_dir/controllers.json"
for deployment in vpa-recommender vpa-updater vpa-admission-controller; do
  kubectl -n kube-system rollout status "deployment/$deployment" --timeout=300s
done
# Admission readiness may precede asynchronous webhook registration.
webhook_ready=false
for attempt in {1..30}; do
  if kubectl get mutatingwebhookconfiguration vpa-webhook-config >/dev/null 2>&1; then webhook_ready=true; break; fi
  sleep 2
done
[[ "$webhook_ready" == true ]] || { echo 'VPA admission webhook was not registered within 60 seconds.' >&2; exit 1; }
kubectl get mutatingwebhookconfiguration vpa-webhook-config
kubectl -n kube-system get pods -l app.kubernetes.io/managed-by=autoops-vpa -o wide
