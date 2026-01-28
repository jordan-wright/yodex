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

output "cloudfront_domain_name" {
  description = "CloudFront domain for the podcast"
  value       = aws_cloudfront_distribution.podcast.domain_name
}

output "acm_validation_records" {
  description = "ACM DNS validation records to add in Cloudflare"
  value = [
    for dvo in aws_acm_certificate.podcast.domain_validation_options : {
      domain = dvo.domain_name
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      value  = dvo.resource_record_value
    }
  ]
}
