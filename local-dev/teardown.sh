#!/usr/bin/env bash
set -euo pipefail

echo "=== Deleting minikube profile: devsecops ==="
minikube delete --profile=devsecops

echo "=== Done ==="
