# Architecture

## Current State (as of 2026-06-07)

**Stg cluster is live.** All security tooling deployed. PR-based promotion to prd is designed but prd cluster not provisioned yet.

| Component | Status | Notes |
|-----------|--------|-------|
| EKS (stg) | ✅ Running | 3x t3.small SPOT, private endpoint, toggle via eks-toggle.sh |
| ArgoCD | ✅ Running | Helm-installed, App-of-Apps from gitops-repo, sync waves, ServerSideApply |
| Kyverno | ✅ 6/7 policies | Chart 3.8.1. verify-image removed (ECR auth limitation, IRSA configured) |
| Employee API | ✅ Deployed | Go app, LoadBalancer service for ZAP scanning |
| Monitoring | ✅ Running | kube-prometheus-stack via direct Helm (not ArgoCD due to CRD conflicts) |
| Grafana | ✅ Running | Access via kubectl port-forward, admin password in Secret |
| CI Pipeline | ✅ 5-stage | Gitleaks → Semgrep → Trivy → Build/Sign/SBOM → Kyverno CLI → Deploy |
| OWASP ZAP | ✅ 68/68 pass | DAST on live stg app via validate.yml |
| DefectDojo | ✅ Ingestion | CI posts Semgrep + Trivy. ZAP posts from validate.yml via Tailscale |
| EKS toggle | ✅ | scripts/eks-toggle.sh stores S3 bucket reference |
| Local testing | ✅ | local-dev/ Minikube setup |
| IRSA for Kyverno | ⚠️ | Role created, SA annotated, env vars injected. ECR auth still 401 in Kyverno |
| verify-image | ⚠️ Removed | Policy deleted from gitops-repo. Cosign signing still works in CI |
| ECR | ✅ | Immutable, KMS-encrypted, SHA-only tags |
| Vault | ❌ | Not configured yet |
| External Secrets | ❌ | Deployed but Degraded (no Vault) |
| prd cluster | ❌ | Terraform layout ready, not provisioned |
| Jira | ❌ | Not configured |

## Stack

| Layer | Tool | Responsibility |
|-------|------|---------------|
| Infrastructure | Terraform | EKS, ECR, IAM, VPC, S3, KMS |
| App | Go + chi + SQLite | Employee Directory API with RBAC |
| Pre-commit | golangci-lint + gitleaks | Catch issues before push |
| Pipeline | GitHub Actions | Lint, SAST, secret scan, container build, vuln scan, policy test |
| CD | ArgoCD | GitOps sync from gitops-repo to EKS |
| Policy Enforcement | Kyverno (Helm chart 3.8.1) | Validates resources at API server level. Sync waves, ServerSideApply |
| Secrets | GitHub Secrets | Pipeline secrets. Vault planned |
| Monitoring | kube-prometheus-stack (direct Helm) | Prometheus, Alertmanager, node-exporter, kube-state-metrics |
| Grafana | On EKS via port-forward | Admin credentials in monitoring/prometheus-grafana Secret |
| Vulnerability Management | DefectDojo (Debian server) | SemGrep + Trivy + ZAP ingestion |
| Ticketing | Jira (planned) | Auto-tickets for Critical/High |
| Connectivity | Tailscale | Home server ↔ EKS ↔ CI |
| Cluster Access | SSM Session Manager | Port forwarding through worker nodes to private EKS endpoint |
| Image Signing | Cosign keyless | Sigstore OIDC, signed in CI Stage 3 |
| DAST | OWASP ZAP | Baseline scan on stg LoadBalancer, runs in gitops-repo validate.yml |

---

## EKS Cluster (stg)

| Setting | Value |
|---------|-------|
| Name | devsecops-cluster |
| Nodes | 3x t3.small (SPOT) |
| K8s version | v1.35.5 |
| Endpoint | endpoint_public_access = false (private-only) |
| Auth mode | API_AND_CONFIG_MAP |
| Storage | encrypted with KMS |
| AMI | AL2023_x86_64_STANDARD |
| Pod limit | 11 pods/node (AWS VPC CNI on t3.small) |

### Cluster Access (SSM Tunnel)

No bastion EC2. Access via SSM port forwarding:

```bash
# Terminal 1
INSTANCE_ID=$(aws ec2 describe-instances --filters "Name=tag:eks:cluster-name,Values=devsecops-cluster" "Name=instance-state-name,Values=running" --query "Reservations[0].Instances[0].InstanceId" --output text)
EKS_ENDPOINT=$(aws eks describe-cluster --name devsecops-cluster --query "cluster.endpoint" --output text | sed 's|https://||')
aws ssm start-session --target "$INSTANCE_ID" --document-name AWS-StartPortForwardingSessionToRemoteHost --parameters "{\"host\":[\"$EKS_ENDPOINT\"],\"portNumber\":[\"443\"],\"localPortNumber\":[\"8443\"]}"

# Terminal 2
aws eks update-kubeconfig --name devsecops-cluster --region ap-southeast-1
CLUSTER_NAME=$(kubectl config view --minify --raw -o json | jq -r '.["current-context"] as $ctx | .contexts[] | select(.name == $ctx) | .context.cluster')
kubectl config set clusters.$CLUSTER_NAME.server https://localhost:8443
kubectl config set clusters.$CLUSTER_NAME.insecure-skip-tls-verify true
kubectl config unset clusters.$CLUSTER_NAME.certificate-authority-data
```

Tunnels die silently when idle. Need to restart and reconfigure when this happens.

### Cost Toggle

```bash
./scripts/eks-toggle.sh stop   # saves ~$75/mo (destroys EKS cluster + nodes)
./scripts/eks-toggle.sh start  # recreates (~15 min)
./scripts/eks-toggle.sh app-url # prints STG_APP_URL after app is deployed
```

Always-running costs: IAM, ECR, S3, VPC, NAT Gateway, KMS (~$38/mo).

---

## Repositories

### app-repo (sycanz/devsecops)

```
employee-api/
├── cmd/server/main.go
├── internal/
│   ├── handler/      (CRUD + health + audit endpoints)
│   ├── middleware/    (JWT auth + RBAC)
│   ├── model/        (Employee struct)
│   ├── store/        (SQLite + audit log)
│   ├── crypto/       (salary encryption)
│   └── server/       (chi router setup)
├── Dockerfile        (golang:1.26-alpine builder → distroless)
├── go.mod / go.sum
├── .golangci.yaml
├── go.work
├── .pre-commit-config.yaml
├── .github/workflows/ci.yml   (5-stage pipeline)
├── scripts/eks-toggle.sh
├── local-dev/                 (Minikube setup for local testing)
│   ├── setup.sh
│   ├── teardown.sh
│   └── config/                (root-app + deployment for minikube)
├── terraform/                 (EKS, ECR, IAM, VPC modules)
├── CONTEXT.md
├── ARCHITECTURE.md
├── LEARNING.md
└── ROADMAP.md
```

### gitops-repo (sycanz/devsecops-gitops)

```
gitops-repo/
├── root-app-stg.yaml          ← Applied to EKS. Multi-source: apps/shared + apps/stg
├── root-app-prd.yaml          ← For future prd cluster
├── apps/
│   ├── shared/                ← Deployed to both clusters
│   │   ├── kyverno.yaml       → Helm chart 3.8.1, sync wave -1, ServerSideApply
│   │   ├── kyverno-policies.yaml → Raw YAML: policies/, sync wave 5, ServerSideApply
│   │   └── external-secrets.yaml → Degraded (no Vault)
│   ├── stg/                   ← Only stg cluster
│   │   └── employee-api.yaml  → Raw YAML: stg/ directory
│   └── prd/                   ← Only prd cluster
│       └── employee-api.yaml
├── stg/
│   └── deployment.yaml        ← Image tag updated by CI pipeline (PLACEHOLDER → ECR SHA)
├── prd/
│   └── deployment.yaml
├── policies/                  ← 6 Kyverno ClusterPolicies
│   ├── drop-all-capabilities.yaml
│   ├── disallow-privileged.yaml
│   ├── disallow-priv-esc.yaml
│   ├── disallow-root.yaml
│   ├── read-only-root-fs.yaml
│   ├── require-pod-probes.yaml
│   └── limit-quota.yaml
├── charts/
│   └── monitoring-values.yaml
├── .zap/rules.tsv
└── .github/workflows/validate.yml  ← Kyverno CLI → OWASP ZAP → DefectDojo
```

**verify-images.yaml removed** — Kyverno's ECR authentication fails (401) even with IRSA configured. Cosign signing still works in CI. Known Kyverno ECR credential-handling limitation.

### App-of-Apps Flow

```
root-app-stg.yaml (applied to argocd namespace)
    │  sources: apps/shared + apps/stg
    │
    ├── apps/shared/kyverno.yaml          (Helm, wave -1)
    ├── apps/shared/kyverno-policies.yaml (Raw YAML, wave 5)
    ├── apps/shared/external-secrets.yaml (Helm, Degraded without Vault)
    └── apps/stg/employee-api.yaml        (Raw YAML, stg/ dir)
```

Sync waves ensure Kyverno CRDs register before policies are applied.

---

## Pipeline Flow (app-repo CI)

```
[Dev machine]
  pre-commit: golangci-lint + gitleaks

[GitHub Actions on push to main/stg]
  Stage 1: Gitleaks
  Stage 2a: SemGrep (SAST) → DefectDojo (via Tailscale)
  Stage 2b: Trivy FS + IaC (SCA) → DefectDojo (via Tailscale)
  Stage 3: Docker build → Trivy image scan → DefectDojo
           → Push to ECR (SHA tag only)
           → Cosign sign → Cosign verify
           → CycloneDX SBOM → GitHub Dependency Graph artifact
  Stage 4: Kyverno CLI validates stg/ + prd/ against policies/
           (verify-image excluded — live Sigstore check unreliable in CLI)
  Stage 5: Clone gitops-repo, update stg/deployment.yaml with ECR SHA tag,
           commit, push → ArgoCD auto-syncs

  Slack: start / each stage failure / success
```

### Gitops-repo PR Validation (validate.yml)

```
PR from stg → main:
  Job: Kyverno CLI validates policies against manifests
  Job: OWASP ZAP baseline scan against STG_APP_URL
       → ZAP report artifact uploaded
       → XML posted to DefectDojo via Tailscale
  Job: Summary gate (passes only if both pass)
```

---

## Kyverno

| Policy | Mode | Status |
|--------|------|--------|
| drop-all-capabilities | Enforce | Active |
| disallow-privileged-containers | Enforce | Active |
| disallow-privilege-escalation | Enforce | Active |
| disallow-root-user | Enforce | Active |
| require-ro-rootfs | Enforce | Active |
| require-pod-probes | Enforce | Active |
| limit-quota | Enforce | Active |
| verify-image | N/A | Removed (ECR auth limitation) |

Kyverno installed via ArgoCD (Helm chart 3.8.1), sync wave -1, ServerSideApply for CRDs.
Kyverno-policies via ArgoCD, sync wave 5, ServerSideApply.

---

## Monitoring

kube-prometheus-stack installed via direct Helm (not ArgoCD — CRD conflicts with chart).
Grafana accessible via port-forward: `kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80`.
Password: `kubectl get secret -n monitoring prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 -d`.

admissionWebhooks.enabled: false — avoids Kyverno conflict with certgen job.

---

## Local Development (Minikube)

```
local-dev/
├── setup.sh          ← One-command: minikube → build image → ArgoCD → deploy
├── teardown.sh       ← minikube delete --profile=devsecops
└── config/
    ├── root-app.yaml      ← ArgoCD root-app for minikube
    └── deployment.yaml    ← Employee API with local image + emptyDir for SQLite
```

---

## Secrets

| Secret | Where | Used by |
|--------|-------|---------|
| AWS_ROLE_ARN | app-repo | CI OIDC → ECR push |
| AWS_REGION | app-repo | CI |
| GITOPS_PAT | app-repo | CI pushes to gitops-repo |
| SEMGREP_APP_TOKEN | app-repo | Semgrep CI scan |
| TS_OIDC_CLIENT_ID / AUDIENCE | both repos | Tailscale OIDC (subject: repo:sycanz/*:*) |
| ONCALL_WEBHOOK_URL | app-repo | Slack notifications |
| DEFECTDOJO_URL / API_KEY | both repos | DefectDojo ingestion |
| STG_APP_URL | gitops-repo | OWASP ZAP target |

---

## Key Decisions (Decisions Made During Implementation)

- SSM Session Manager replaces bastion EC2 (no SSH keys, no public IP)
- Kyverno via ArgoCD Helm chart 3.8.1 (not direct helm install) with sync waves + ServerSideApply
- Monitoring via direct Helm (not ArgoCD) due to kube-prometheus-stack CRD + ArgoCD conflicts
- verify-image removed due to Kyverno ECR credential handling limitation (IRSA configured but 401 persists)
- t3.small with 3 nodes (up from 2) due to pod density limits (11 pods/node on VPC CNI)
- ServerSideApply for all CRD-heavy apps (Kyverno, monitoring) to bypass 256KB annotation limits
- Employee API uses LoadBalancer service type for OWASP ZAP DAST access
- Tailscale OIDC subject: repo:sycanz/*:* covers both repos

---

## Future / Stretch

- Vault integration (secret management + ESO sync)
- verify-image re-enablement (fix Kyverno ECR auth)
- Jira auto-ticketing from DefectDojo findings
- prd cluster provisioning
- PR-based stg → prd promotion
- ArgoCD Rollouts for canary deployments
- CIS EKS Benchmark scanning via kube-bench
- Grafana on home server scraping cluster Prometheus via Tailscale
- Cost dashboards in Grafana
- Deep-dive fine-tuning: SemGrep custom rules, Trivy ignore policies
