#!/usr/bin/env bash
set -euo pipefail

CLUSTER="devsecops-cluster"
REGION="ap-southeast-1"
NAMESPACE_VAULT="vault"
NAMESPACE_ESO="external-secrets"

echo "=== 1. Ensuring kubeconfig is current ==="
aws eks update-kubeconfig --name "$CLUSTER" --region "$REGION"

echo "=== 2. Creating namespaces ==="
kubectl create namespace "$NAMESPACE_VAULT" --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace "$NAMESPACE_ESO" --dry-run=client -o yaml | kubectl apply -f -

echo "=== 3. Installing Helm (if missing) ==="
if ! command -v helm &>/dev/null; then
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

echo "=== 4. Deploying Vault ==="
helm repo add hashicorp https://helm.releases.hashicorp.com 2>/dev/null || true
helm upgrade --install vault hashicorp/vault \
  --namespace "$NAMESPACE_VAULT" \
  --set server.dev.enabled=true \
  --set "server.extraArgs=-dev-root-token-id=root" \
  --wait

echo "=== 5. Deploying External Secrets Operator ==="
helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
helm upgrade --install external-secrets external-secrets/external-secrets \
  --namespace "$NAMESPACE_ESO" \
  --wait

echo "=== 6. Configuring Vault ==="
# Port-forward to Vault
kubectl port-forward -n "$NAMESPACE_VAULT" svc/vault 8200:8200 &
PF_PID=$!
sleep 3

export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"

# Enable KV v2 secrets engine
vault secrets enable -path=secret kv-v2 2>/dev/null || true

# Store app secrets
vault kv put secret/app/employee-directory \
  ADMIN_TOKEN="admin-$(openssl rand -hex 16)" \
  MANAGER_TOKEN="manager-$(openssl rand -hex 16)" \
  EMPLOYEE_TOKEN="employee-$(openssl rand -hex 16)" \
  JWT_SECRET="jwt-$(openssl rand -hex 32)" \
  SALARY_ENCRYPTION_KEY="enc-$(openssl rand -hex 32)"

echo "  Secrets written to Vault path: secret/app/employee-directory"

# Enable Kubernetes auth
vault auth enable kubernetes 2>/dev/null || true

# Get the ESO service account token
ESO_SA_JWT=$(kubectl get secret -n "$NAMESPACE_ESO" \
  $(kubectl get serviceaccount -n "$NAMESPACE_ESO" external-secrets -o jsonpath='{.secrets[0].name}') \
  -o jsonpath='{.data.token}' | base64 -d)

# Get CA cert and API server URL
K8S_CA_CERT=$(kubectl get secret -n "$NAMESPACE_ESO" \
  $(kubectl get serviceaccount -n "$NAMESPACE_ESO" external-secrets -o jsonpath='{.secrets[0].name}') \
  -o jsonpath='{.data.ca\.crt}' | base64 -d)

K8S_HOST=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

# Configure Kubernetes auth
vault write auth/kubernetes/config \
  token_reviewer_jwt="$ESO_SA_JWT" \
  kubernetes_host="$K8S_HOST" \
  kubernetes_ca_cert="$K8S_CA_CERT"

# Create a policy for reading app secrets
vault policy write employee-directory - <<'POL'
path "secret/data/app/employee-directory" {
  capabilities = ["read"]
}
POL

# Create a role linking the ESO service account to the policy
vault write auth/kubernetes/role/employee-directory \
  bound_service_account_names="external-secrets" \
  bound_service_account_namespaces="$NAMESPACE_ESO" \
  policies="employee-directory" \
  ttl="1h"

# Kill the port-forward
kill "$PF_PID" 2>/dev/null || true

echo "=== 7. Creating SecretStore and ExternalSecret ==="
kubectl apply -f k8s/external-secrets/secret-store.yaml
kubectl apply -f k8s/external-secrets/external-secret.yaml

echo "=== 8. Deploying the app ==="
kubectl apply -f k8s/app/deployment.yaml

echo ""
echo "=== Done! ==="
echo "App secrets are now managed by Vault and synced to K8s via External Secrets Operator."
echo "Verify with: kubectl get secret app-secrets -n employee-api"
