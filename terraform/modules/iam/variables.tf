variable "github_repository" {
  description = "The GitHub repository allowed to push to ECR (e.g., 'username/repo')"
  type        = string
  default     = "*" # Caution: '*' allows any repo in your GitHub profile if using a generic condition.
}
