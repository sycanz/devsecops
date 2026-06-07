data "aws_caller_identity" "current" {}

resource "aws_kms_key" "eks" {
  description             = "EKS secret encryption key"
  deletion_window_in_days = 7
  enable_key_rotation     = true
}

resource "aws_eks_cluster" "main" {
  name     = "devsecops-cluster"
  role_arn = var.cluster_role_arn

  vpc_config {
    subnet_ids = var.private_subnets
    endpoint_private_access = true
    endpoint_public_access  = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks.arn
    }
    resources = ["secrets"]
  }

  access_config {
    authentication_mode                         = "API_AND_CONFIG_MAP"
    bootstrap_cluster_creator_admin_permissions = true
  }
}

resource "aws_eks_node_group" "main" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "standard-nodes"
  node_role_arn   = var.node_group_role_arn
  subnet_ids      = var.private_subnets

  scaling_config {
    desired_size = 2
    max_size     = 3
    min_size     = 1
  }

  update_config {
    max_unavailable = 1
  }

  instance_types = ["t3.small"]

  # Standard hardened configuration  
  ami_type = "AL2023_x86_64_STANDARD"
  capacity_type = "SPOT"
}

# Allow the node IAM role to run kubectl from the node (via SSM)
resource "aws_eks_access_entry" "nodes" {
  cluster_name  = aws_eks_cluster.main.name
  principal_arn = var.node_group_role_arn
  type          = "STANDARD"
}

resource "aws_eks_access_policy_association" "nodes_admin" {
  depends_on = [aws_eks_access_entry.nodes]

  cluster_name  = aws_eks_cluster.main.name
  principal_arn = var.node_group_role_arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
  access_scope {
    type = "cluster"
  }
}
