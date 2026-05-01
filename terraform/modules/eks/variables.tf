variable "vpc_id" {
  description = "The VPC ID where the cluster will be created"
  type        = string
}

variable "private_subnets" {
  description = "A list of private subnet IDs for the EKS nodes"
  type        = list(string)
}

variable "cluster_role_arn" {
  description = "The ARN of the IAM role for the EKS cluster"
  type        = string
}

variable "node_group_role_arn" {
  description = "The ARN of the IAM role for the EKS node group"
  type        = string
}
