output "bucket_name" {
  description = "S3 bucket for episodes"
  value       = aws_s3_bucket.episodes.bucket
}

output "region" {
  description = "AWS region"
  value       = var.region
}

output "role_arn" {
  description = "IAM role ARN for GitHub Actions"
  value       = aws_iam_role.github_actions.arn
}

output "s3_prefix" {
  description = "S3 prefix used for uploads"
  value       = local.full_prefix
}
