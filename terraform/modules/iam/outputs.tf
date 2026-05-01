output "eks_cluster_role_arn" {
  value = aws_iam_role.eks_cluster.arn
}

output "eks_node_group_role_arn" {
  value = aws_iam_role.eks_nodes.arn
}

output "github_actions_role_arn" {
  value = aws_iam_role.github_actions.arn
}
