resource "aws_ecr_repository" "app" {
  name                 = "devsecops-app"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
  }

  tags = {
    Project = "devsecops"
  }
}

output "repository_url" {
  value = aws_ecr_repository.app.repository_url
}
