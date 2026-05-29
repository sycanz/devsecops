# Architecture

## Stack

| Layer | Tool | Responsibility |
|-------|------|---------------|
| Infrastructure | Terraform | EKS, ECR, IAM, networking, S3, DynamoDB |
| App | Go + chi + SQLite | Employee Directory API with RBAC |
| Pre-commit | golangci-lint + gitleaks + go test | Catch issues before push |
| Pipeline | GitHub Actions | Lint, SAST, secret scan, container build, vuln scan, policy test |
| CD | ArgoCD | GitOps sync from gitops-repo to EKS |
| Policy Enforcement | Kyverno | Validates all resources at API server level |
| Secrets | Vault (Debian server) + External Secrets Operator | Centralized secrets management |
| Monitoring | kube-prometheus-stack on EKS + Grafana (home server) | Cluster + app observability |
| Vulnerability Management | DefectDojo (Debian server) | Centralizes SemGrep + Trivy findings, dedup, trending |
| Ticketing | Jira (via API) | Auto-creates tickets for Critical/High findings with business risk context |
| Connectivity | Tailscale | Private network between home server, EKS, and CI |

---

## Terraform Structure

```
terraform/
├── modules/
│   ├── vpc/
│   ├── eks/
│   ├── iam/
│   └── ecr/
├── environments/
│   ├── stg/
│   │   ├── main.tf        (calls modules with stg vars)
│   │   ├── backend.tf       (stg state bucket + DynamoDB)
│   │   └── terraform.tfvars (smaller node sizes)
│   └── prd/
│       ├── main.tf          (calls modules with prd vars)
│       ├── backend.tf       (prd state bucket + DynamoDB)
│       └── terraform.tfvars (larger node sizes)
```

Separate state backends per environment. Separate EKS clusters per environment.

---

## Repositories

### app-repo
```
employee-api/
├── cmd/server/main.go
├── internal/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── store/
│   ├── crypto/
│   └── server/
├── k8s/
│   └── base/                (canonical manifests)
├── Dockerfile
├── go.mod
└── .github/workflows/
    └── ci.yml
```

### gitops-repo (separate)
```
gitops-repo/
├── stg/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   └── kustomization.yaml
└── prd/
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    └── kustomization.yaml
```

ArgoCD instances each watch their respective directory.

---

## Branch Strategy

| Branch | Trigger | Pipeline | Deploys to |
|--------|---------|----------|------------|
| `feature/*` | Push | Lint + test only (fast) | None |
| `staging` | Push | Full scan + build + test | stg-cluster |
| `main` | PR merge (from staging) | Full scan + build + test | prd-cluster |

---

## Pipeline Flow

```
[Dev machine]
  git commit hook: golangci-lint + gitleaks + go test ./...

[GitHub Actions — push to staging or main]
  Stage 1: GitLeaks (belt-and-suspenders)
  Stage 2: SemGrep (SAST) + Trivy fs on go.mod (SCA)
  Stage 3: kyverno apply k8s/ --policy k8s/kyverno/ (pre-deploy validation)
  Stage 4: Docker build → Trivy image scan → SBOM (CycloneDX format)
  Stage 5: Update gitops-repo with new image tag → ArgoCD syncs

  Stage 6: Import scan results → DefectDojo (SemGrep JSON + Trivy JSON)
  Stage 7: If Critical/High → auto-create Jira ticket with DefectDojo link + business risk

  Slack notifications on: start / success / failure
```

---

## EKS Clusters

| Environment | Nodes | Kyverno mode | Purpose |
|-------------|-------|-------------|---------|
| stg | 1-2 x t3.micro (spot) | Enforce | Test pipeline + policy changes |
| prd | 2-3 x t3.medium (spot) | Audit → Enforce (gradual) | Production workloads |

Kyverno installed cluster-wide in `kyverno-system` namespace on both clusters.

---

## Secrets Flow

```
GitHub Actions (OIDC auth)
  → Vault (on Debian server via Tailscale)
  → Short-lived secrets at pipeline runtime

Vault
  ├── stg/
  │   ├── database/     (staging DB creds)
  │   ├── jwt/          (staging JWT signing key)
  │   └── k8s/          (staging cluster secrets)
  └── prd/
      ├── database/     (production DB creds, different)
      ├── jwt/
      └── k8s/

External Secrets Operator (on both EKS clusters)
  → Syncs Vault secrets into K8s Secret objects
  → Pods consume normally (env vars or volume mounts)
```

Vault runs on the Debian home server alongside Grafana. Both EKS clusters reach it via Tailscale.

---

## Monitoring Architecture

```
[EKS stg-cluster]
  kube-prometheus-stack (Prometheus + kube-state-metrics + node-exporter)

[EKS prd-cluster]
  kube-prometheus-stack (Prometheus + kube-state-metrics + node-exporter)

[Debian Home Server]
  cadvisor + node-exporter (host-level metrics)
  Grafana ← Prometheus data sources (stg + prd + local) via Tailscale
  Alertmanager (optional, can forward to Slack)
  DefectDojo ← ingests SemGrep + Trivy results via API from pipeline
```

Grafana dashboards track:
- Pipeline health (deployment frequency, failure rate)
- Cluster health (node status, pod restarts, resource usage)
- Application health (request latency, error rate, throughput)
- Security posture (Kyverno policy violations, container vulnerabilities)

---

## Deploy Flow (Target — after Phase 2)

```
1. terraform apply -environments/stg  → provisions stg-EKS + stg-ECR
2. terraform apply -environments/prd  → provisions prd-EKS + prd-ECR
3. Push to staging branch
   → Pipeline runs full scan → pushes image → updates gitops-repo/stg/
   → ArgoCD on stg-cluster detects drift → syncs
4. PR staging → main (after validation on stg)
   → Pipeline runs full scan again → pushes image → updates gitops-repo/prd/
   → ArgoCD on prd-cluster detects drift → syncs
5. Kyverno evaluates every resource at admission time on both clusters
```

---

## Vulnerability Management (DefectDojo)

DefectDojo runs on the Debian home server (Docker), accessible via Tailscale. It aggregates scan results from the pipeline into a single dashboard.

```
Pipeline
  ├── SemGrep JSON ──┐
  ├── Trivy FS JSON ──┤──→ DefectDojo API → dedup → trending → dashboard
  └── Trivy Image JSON ┘
```

Key features used:
- **Auto-import**: Pipeline pushes JSON reports to DefectDojo API after each scan
- **Deduplication**: Same CVE across multiple scans → grouped
- **Severity filtering**: Only Critical/High trigger Jira tickets
- **Product segmentation**: Separate products for stg vs prd findings

## Ticket & Tackle (Jira Integration)

On pipeline failure with Critical/High findings:

```
SemGrep/Trivy find Critical vuln
  → DefectDojo ingests the report
  → Pipeline script queries DefectDojo API for new Critical/High findings
  → Creates Jira ticket with:
      Summary: "[Security] {CVE-ID} in {target}"
      Description: {business risk summary}
      Link: {DefectDojo finding URL}
      Labels: security, critical
  → Ticket key returned and logged in pipeline output
```

This is implemented as a post-scan pipeline step (not a full-fledged automation — keeps it simple for a portfolio project).

## Kyverno Rollout Strategy

| Phase | Mode | Duration |
|-------|------|----------|
| 1 | Audit | Week 1-2 (observe violations) |
| 2 | Enforce (stg only) | Week 2-3 (test impact) |
| 3 | Enforce (prd, non-critical policies) | Week 3-4 |
| 4 | Enforce (all policies) | After validation |

---

## Future / Stretch

- Vault Agent Injector (sidecar) for zero-trust secrets (upgrade from ESO)
- ArgoCD Rollouts for canary deployments
- CIS EKS Benchmark scanning via kube-bench
- Cost dashboards in Grafana
- Deep-dive fine-tuning: SemGrep custom rules, Trivy ignore policies, Kyverno policy library
- Jira automation → Slack notification on ticket creation
