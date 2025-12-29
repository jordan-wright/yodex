variable "region" {
  type        = string
  description = "AWS region for resources"
  default     = "us-west-2"
}

variable "bucket_name" {
  type        = string
  description = "S3 bucket name for episodes"
  default     = "yodex-episodes"
}

variable "s3_prefix" {
  type        = string
  description = "S3 key prefix for episodes"
  default     = "yodex"
}

variable "github_owner" {
  type        = string
  description = "GitHub org/user owning the repo"
}

variable "github_repo" {
  type        = string
  description = "GitHub repo name"
}

variable "github_branch" {
  type        = string
  description = "Branch allowed to assume the role"
  default     = "main"
}
