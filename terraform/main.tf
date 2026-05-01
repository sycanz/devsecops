provider "aws" {
  region = "ap-southeast-1"
}

module "networking" {
  source = "./modules/networking"
}

module "iam" {
  source = "./modules/iam"

  # Replace this with your actual GitHub repository (e.g. "sycanz/devsecops")
  github_repository = "sycanz/devsecops"
}

# module "eks" {
#   source = "./modules/eks"
# 
#   vpc_id          = module.networking.vpc_id
#   private_subnets = module.networking.private_subnets
#   
#   cluster_role_arn    = module.iam.eks_cluster_role_arn
#   node_group_role_arn = module.iam.eks_node_group_role_arn
# }

module "ecr" {
  source = "./modules/ecr"
}
