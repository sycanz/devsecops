# DevSecOps Portfolio Project

Cloud-native security governance project with IaC provisioning, CI/CD pipeline with security scanning, and Kubernetes policy enforcement.

## Language

**EKS (Elastic Kubernetes Service)**:
AWS-managed Kubernetes cluster for hosting workloads.
_Avoid_: plain Kubernetes, K8s

**Kyverno**:
Policy-as-code engine for Kubernetes. Validates and enforces security policies on manifests and resources.
_Avoid_: OPA Gatekeeper (alternative)

**Pipeline**:
GitHub Actions workflow that builds, scans, and deploys the application.
_Avoid_: CI/CD, workflow

**Trivy**:
Multi-purpose security scanner. Scans container images, filesystems, and IaC configurations for vulnerabilities.
_Avoid_: Clair, Anchore (alternatives)

**golangci-lint**:
Go linter runner. Catches bugs, style issues, and security anti-patterns in pre-commit.
_Avoid_: staticcheck alone, revive alone

**SemGrep**:
SAST tool for catching code-level vulnerabilities and anti-patterns in Go and IaC files.
_Avoid_: SonarQube (alternative, heavier), CodeQL (alternative, GitHub-only)

**GitLeaks**:
Static analysis tool for detecting hardcoded secrets in git repositories.
_Avoid_: git-secrets, truffleHog (alternatives)

**ECR (Elastic Container Registry)**:
AWS-managed container registry for storing Docker images.
_Avoid_: Docker Hub, ACR

**Terraform**:
Infrastructure-as-code tool for provisioning AWS resources.
_Avoid_: CloudFormation, Pulumi (alternatives)

**Vault**:
Secret management tool for centralized secrets storage and rotation.
_Avoid_: AWS Secrets Manager, SSM Parameter Store (for production)

**Employee Directory API**:
Go-based app managing employee records (name, email, department, role, salary). Stores PII, scoped under GDPR controls.
_Avoid_: sample app, test workload, the app

**GDPR (General Data Protection Regulation)**:
EU regulation governing personal data. Used as the compliance reference for the Employee Directory API.
_Avoid_: HIPAA, PCI-DSS (for this project's scope)

**SOC2**:
Audit framework for service organizations. Referenced for controls (access control, monitoring, data disposal).
_Avoid_: ISO 27001 (alternative, broader)

**External Secrets Operator**:
Kubernetes operator that syncs secrets from external providers (Vault) into the cluster.
_Avoid_: Sealed Secrets, SOPS (alternatives)

**ArgoCD**:
GitOps deployment tool. Watches gitops-repo and syncs manifests to EKS automatically.
_Avoid_: Flux (alternative), manual kubectl

**SBOM (Software Bill of Materials)**:
Machine-readable inventory of all dependencies in the container image. Generated post-build in CycloneDX format.
_Avoid_: SPDX (alternative)

**kube-prometheus-stack**:
Helm chart bundling Prometheus, Alertmanager, kube-state-metrics, node-exporter, and Grafana dashboards. Deployed on EKS for cluster monitoring.
_Avoid_: Prometheus Operator (predecessor name)

**Tailscale**:
WireGuard-based VPN for private networking between home server (Vault, Grafana) and EKS clusters.
_Avoid_: OpenVPN, WireGuard manual config

**DefectDojo**:
Open-source vulnerability management platform. Ingests SemGrep and Trivy reports, deduplicates findings, and provides trending dashboards.
_Avoid_: Snyk Dashboard (paid), WhiteSource (paid)

**Jira**:
Issue tracking system. Used for "Ticket & Tackle" — auto-creating tickets for Critical/High security findings with DefectDojo links and business risk context.
_Avoid_: Linear, Trello (for this workflow)

**OWASP ZAP**:
DAST (Dynamic Application Security Testing) tool. Scans running applications for runtime vulnerabilities like XSS, CSRF, SQL injection.
_Avoid_: Burp Suite (alternative, paid)

**Cosign**:
Container image signing tool. Signs images with keyless OIDC-based signatures via Sigstore.
_Avoid_: Docker Content Trust (alternative, less flexible)

**SSM Session Manager**:
AWS Systems Manager feature for secure remote access to EC2 instances without SSH or public IPs. Used for port forwarding to reach the private EKS endpoint from local machine.
_Avoid_: bastion host, SSH tunnel

**IRSA (IAM Roles for Service Accounts)**:
EKS feature that associates IAM roles with Kubernetes service accounts via OIDC. Used to give Kyverno ECR read access for image verification.
_Avoid_: kube2iam, hardcoded credentials

**LoadBalancer (AWS NLB)**:
Network Load Balancer created when a Kubernetes Service type=LoadBalancer is created on EKS. Exposes the Employee API publicly for OWASP ZAP scanning.
_Avoid_: ClusterIP (internal-only)

## Relationships

- **Pipeline** lints (golangci-lint) → scans (GitLeaks, SemGrep, Trivy) → builds image → scans image → signs (Cosign) → validates manifests (Kyverno CLI) → pushes to ECR → updates gitops-repo
- **ArgoCD** watches gitops-repo → syncs manifests to EKS
- **Kyverno** runs on EKS to enforce security policies on deployed resources
- **Terraform** provisions EKS, ECR, networking, and IAM resources (separate for stg/prd)
- **Trivy** scans code dependencies, IaC configs, and container images at various pipeline stages
- **GitLeaks** scans commits for leaked secrets before pipeline proceeds
- **SemGrep** scans Go and IaC code for vulnerabilities and anti-patterns before build
- **Vault** serves secrets to pipeline (via OIDC) and to EKS clusters (via ESO) over Tailscale
- **kube-prometheus-stack** on EKS exposes metrics → **Grafana** on home server scrapes via Tailscale
- **DefectDojo** ingests SemGrep and Trivy findings from pipeline → generates trend reports
- **Jira** tickets auto-created for Critical/High findings from **DefectDojo** (Ticket & Tackle)
- **OWASP ZAP** runs DAST on deployed EKS app → catches runtime vulnerabilities missed by SAST/SCA
- **Cosign** signs images in pipeline → Kyverno verifyImages policy enforces only signed images at cluster admission
- **GitOps CI** (gitops-repo) runs Kyverno CLI + OWASP ZAP on PRs before promotion to prd

## Example dialogue

> **Dev:** "When we deploy to EKS, does Kyverno block the deployment if a container runs as root?"
> **Security engineer:** "Yes — our policy enforces `runAsNonRoot: true` and will reject pods that violate it. We test policies in the pipeline before deployment, so issues are caught early."

## Flagged ambiguities

- "app" — sometimes refers to the sample workload (nginx), sometimes refers to the security tooling (Kyverno, scanners). Resolved: clarify "sample app" vs "security tooling" in context.

## Key Decisions

- **Terraform state**: Use S3 backend with DynamoDB locking (not local); S3 bucket encrypted with customer-managed KMS key
- **Terraform layout**: environments/ (stg, prd) calling shared modules/ (not workspaces)
- **EKS clusters**: Two separate clusters (stg + prd), not namespaces within one
- **GitOps**: Separate gitops-repo with stg/ and prd/ directories; ArgoCD watches each
- **Kyverno installation**: Cluster-wide in kyverno-system namespace on both clusters
- **Kyverno mode**: Audit on prd first, flip to Enforce after stg validation
- **Kyverno in pipeline**: kyverno apply dry-run as Stage 4 — validates manifests pre-deployment (shift-left)
- **Secrets for pipeline**: GitHub Secrets for bootstrapping; Vault with OIDC auth as target
- **Vault location**: Debian home server, accessible via Tailscale
- **Secrets to cluster**: External Secrets Operator syncs Vault → K8s Secrets
- **Public repo**: Keep ECR URLs, AWS account IDs, cluster endpoints as placeholders or in secrets
- **Compliance framework**: GDPR as primary reference (SOC2 secondary). Not switching to PCI-DSS/HIPAA.
- **App scope**: Employee Directory API with PII (email) and sensitive data (salary). No government IDs.
- **Language**: Go (not Python — stronger alignment with cloud-native ecosystem)
- **Pre-commit**: golangci-lint + gitleaks + go test (not ruff)
- **SAST tool**: SemGrep for code-level scanning (before build stage)
- **Container scanner**: Trivy for both filesystem (SCA) and image scanning
- **SBOM**: Generated post-build with trivy image --format cyclonedx
- **CD approach**: ArgoCD (not kubectl apply via pipeline)
- **Image signing**: Cosign keyless signing via Sigstore; Kyverno verifyImages enforces at admission
- **Monitoring**: kube-prometheus-stack on EKS, Grafana on home server
- **Connectivity**: Tailscale between home server ↔ EKS ↔ CI (when needed)
- **Slack notifications**: Pipeline start / success / failure via webhook
- **Vulnerability management**: DefectDojo on Debian server, ingests SemGrep + Trivy reports
- **Ticketing**: Jira tickets created for Critical/High findings with DefectDojo link + business risk

## Key Decisions (Made During Implementation)

- **Cluster access**: SSM Session Manager port forwarding replaces bastion EC2 (no SSH keys, no public IP)
- **Kyverno management**: ArgoCD via Helm chart 3.8.1 with sync waves (wave -1 for core, wave 5 for policies) + ServerSideApply for large CRDs
- **Monitoring**: kube-prometheus-stack installed via direct Helm (not ArgoCD) due to CRD annotation size conflicts with ArgoCD's Helm handling
- **verify-image**: Removed from Kyverno policies. IRSA configured for ECR access but Kyverno's ECR credential handling returns 401. Cosign signing still works in CI
- **Pod density**: t3.small (11 pods/node limit on AWS VPC CNI), scaled to 3 nodes to fit all tooling
- **ServerSideApply**: Used for all CRD-heavy apps (Kyverno, monitoring) to bypass Kubernetes 256KB annotation limits
- **ZAP scanning**: Employee API exposed via LoadBalancer type Service so OWASP ZAP can reach it from GitHub Actions runner
- **Tailscale OIDC**: Subject set to repo:sycanz/*:* to cover both app-repo and gitops-repo