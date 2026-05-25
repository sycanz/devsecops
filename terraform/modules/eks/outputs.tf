output "cluster_name" {
  value = aws_eks_cluster.main.name
}

output "endpoint" {
  value = aws_eks_cluster.main.endpoint
}

output "certificate_authority" {
  value = aws_eks_cluster.main.certificate_authority[0].data
}

output "cluster_security_group_id" {
  value = aws_eks_cluster.main.vpc_config[0].cluster_security_group_id
}

output "oidc_provider_url" {                                                                                                                 
  value = aws_eks_cluster.main.identity[0].oidc[0].issuer                                                                                    
}

output "oidc_provider_arn" {
  value = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${trimprefix(aws_eks_cluster.main.identity[0].oidc[0]    
.issuer, "https://")}"                                                                                                                        
}
