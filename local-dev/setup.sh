#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/.." && pwd)"
GITOPS_REPO="/home/sycanz/projects/devsecops-gitops"
PROFILE="devsecops"

echo "=== Step 1: Start Minikube ==="
minikube start --driver=docker --profile="$PROFILE" --cpus=4 --memory=8g

echo ""
echo "=== Step 2: Point Docker to Minikube ==="
eval $(minikube docker-env --profile="$PROFILE")

echo ""
echo "=== Step 3: Build app image ==="
docker build -t employee-api:local "$ROOT"

echo ""
echo "=== Step 4: Install ArgoCD ==="
helm repo add argo https://argoproj.github.io/argo-helm 2>/dev/null || true
helm install argocd argo/argo-cd -n argocd --create-namespace --wait

echo ""
echo "=== Step 5: Apply root-app (Kyverno wave -1, policies wave 5, monitoring) ==="
kubectl apply -f "$DIR/config/root-app.yaml"

echo ""
echo "=== Step 6: Wait for Kyverno (sync wave ordering handles the race) ==="
echo "(checking every 15s, timeout 5 min)"
for i in $(seq 1 20); do
  READY=$(kubectl get pods -n kyverno-system 2>/dev/null | grep -c "Running" || true)
  if [ "$READY" -ge 3 ]; then
    echo "$READY/4 Kyverno pods running!"
    break
  fi
  echo "  waiting... ($i/20, $READY/4 running)"
  sleep 15
done

echo ""
echo "=== Step 7: Deploy employee-api ==="
sleep 5
kubectl apply -f "$DIR/config/deployment.yaml"

echo ""
echo "=== Done ==="
echo "Check pods:  kubectl get pods -A"
echo "Check ArgoCD: kubectl get applications -n argocd"
echo "Test API:    kubectl port-forward -n employee-api svc/employee-directory 8000:8000"
echo ""
echo "IMPORTANT: gitops-repo changes must be pushed before fresh run:"
echo "  cd $GITOPS_REPO && git add -A && git commit -m 'kyverno: sync waves + ServerSideApply' && git push"
echo ""
echo "Teardown: minikube delete --profile=$PROFILE"
