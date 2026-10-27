#!/usr/bin/env bash
# Install a bundle rendered by prepare.py. Requires an already reviewed context.
set -euo pipefail
bundle=${1:?Usage: install.sh <prepared-bundle-directory>}
profile=bounded-lab-20261002
for file in serving-crds.yaml serving-core.yaml kourier.yaml review.json; do
  test -s "$bundle/$file"
done
kubectl version -o json | python3 -c 'import json,sys; v=json.load(sys.stdin)["serverVersion"]; ok=int(v["major"])==1 and int("".join(c for c in v["minor"] if c.isdigit()))>=34; sys.exit(0 if ok else "Knative 1.23 requires Kubernetes >=1.34")'
kubectl get nodes -l kubernetes.io/role=node -o json | python3 -c 'import json,sys; sys.exit(0 if json.load(sys.stdin)["items"] else "No worker matching kubernetes.io/role=node")'
for ns in knative-serving kourier-system; do
  existing=$(kubectl get namespace "$ns" --ignore-not-found -o 'jsonpath={.metadata.labels.autoops\.io/install-profile}')
  if kubectl get namespace "$ns" >/dev/null 2>&1 && [[ "$existing" != "$profile" ]]; then
    echo "Refusing to change existing unowned namespace: $ns" >&2
    exit 1
  fi
done
# Server-side apply avoids the last-applied annotation limit on large CRDs.
kubectl apply --server-side --field-manager=autoops-knative-install -f "$bundle/serving-crds.yaml"
kubectl wait --for=condition=Established crd -l autoops.io/install-profile="$profile" --timeout=120s
kubectl apply --server-side --field-manager=autoops-knative-install -f "$bundle/serving-core.yaml"
for deployment in activator autoscaler controller webhook; do
  kubectl -n knative-serving rollout status deployment/"$deployment" --timeout=240s
done
kubectl apply --server-side --field-manager=autoops-knative-install -f "$bundle/kourier.yaml"
kubectl -n knative-serving rollout status deployment/net-kourier-controller --timeout=240s
kubectl -n kourier-system rollout status deployment/3scale-kourier-gateway --timeout=240s
kubectl -n knative-serving get pods -o wide
kubectl -n kourier-system get pods,service -o wide
