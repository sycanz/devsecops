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

**Ruff**:
Fast Python linter and formatter.
_Avoid_: flake8, pylint (alternatives)

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
FastAPI-based app managing employee records (name, email, department, role, salary). Stores PII, scoped under GDPR controls.
_Avoid_: sample app, test workload, the app

**GDPR (General Data Protection Regulation)**:
EU regulation governing personal data. Used as the compliance reference for the Employee Directory API.
_Avoid_: HIPAA, PCI-DSS (for this project's scope)

**SOC2**:
Audit framework for service organizations. Referenced for controls (access control, monitoring, data disposal).
_Avoid_: ISO 27001 (alternative, broader)

**External Secrets Operator**:
Kubernetes operator that syncs secrets from external providers (GitHub, AWS) into the cluster.
_Avoid_: Sealed Secrets, SOPS (alternatives)

## Relationships

- **Pipeline** builds Docker image → pushes to **ECR** → deploys to **EKS**
- **Kyverno** runs on **EKS** to enforce security policies on deployed resources
- **Terraform** provisions **EKS**, **ECR**, networking, and IAM resources
- **Trivy** scans code dependencies, filesystem, and Docker images at various pipeline stages
- **GitLeaks** scans commits for leaked secrets before pipeline proceeds

## Example dialogue

> **Dev:** "When we deploy to **EKS**, does **Kyverno** block the deployment if a container runs as root?"
> **Security engineer:** "Yes — our policy enforces `runAsNonRoot: true` and will reject pods that violate it. We test policies in the pipeline before deployment, so issues are caught early."

## Flagged ambiguities

- "app" — sometimes refers to the sample workload (nginx), sometimes refers to the security tooling (Kyverno, scanners). Resolved: clarify "sample app" vs "security tooling" in context.

## Key Decisions

- **Terraform state**: Use S3 backend with DynamoDB locking (not local)
- **Kyverno installation**: Cluster-wide in `kyverno-system` namespace
- **Kyverno mode**: Start with `enforce: false` (audit), move to `enforce: true` after testing
- **Secrets for pipeline**: GitHub Secrets (document Vault as production upgrade)
- **Pipeline pre-deployment**: Add Kyverno policy validation step using `kyverno apply`
- **Public repo**: Keep ECR URLs, AWS account IDs, cluster endpoints as placeholders or in secrets
- **Compliance framework**: GDPR as primary reference (SOC2 secondary). Not switching to PCI-DSS/HIPAA.
- **App scope**: Employee Directory API with PII (email) and sensitive data (salary). No government IDs.