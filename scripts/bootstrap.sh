#! /usr/bin/bash
cd ../terraform
terraform apply -auto-approve

aws eks update-kubeconfig --name devsecops-cluster --region ap-southeast-1

# helm repo add kyverno https://kyverno.github.io/kyverno/
# helm repo update
# helm upgrade --install kyverno kyverno/kyverno -n kyverno --create-namespace

# kubectl apply -f ../k8s/policies/
#
# helm repo add hashicorp https://helm.releases.hashicorp.com
# helm repo update
