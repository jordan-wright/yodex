provider "aws" {
  region = var.region
}

locals {
  full_prefix = trim(var.s3_prefix, "/")
}

# Bucket to store daily episodes and latest pointers.
resource "aws_s3_bucket" "episodes" {
  bucket = var.bucket_name

  tags = {
    Project = "yodex"
  }
}

# Keep older versions when overwriting latest objects.
resource "aws_s3_bucket_versioning" "episodes" {
  bucket = aws_s3_bucket.episodes.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "episodes" {
  bucket = aws_s3_bucket.episodes.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Allow public object reads only via the bucket policy below.
resource "aws_s3_bucket_public_access_block" "episodes" {
  bucket = aws_s3_bucket.episodes.id

  block_public_acls       = true
  ignore_public_acls      = true
  block_public_policy     = false
  restrict_public_buckets = false
}

data "aws_iam_policy_document" "episodes_public" {
  statement {
    sid     = "PublicReadEpisodes"
    effect  = "Allow"
    actions = ["s3:GetObject"]
    resources = [
      "${aws_s3_bucket.episodes.arn}/${local.full_prefix}/latest/*"
    ]

    principals {
      type        = "*"
      identifiers = ["*"]
    }
  }
}

# Public read only for the latest episode objects.
resource "aws_s3_bucket_policy" "episodes_public" {
  bucket = aws_s3_bucket.episodes.id
  policy = data.aws_iam_policy_document.episodes_public.json

  depends_on = [aws_s3_bucket_public_access_block.episodes]
}

# GitHub OIDC provider for Actions (thumbprint is now ignored by AWS for GitHub OIDC).
resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = ["sts.amazonaws.com"]
}

# Read the GitHub Actions OIDC cert chain for the thumbprint.

# IAM role assumed by GitHub Actions via OIDC.
resource "aws_iam_role" "github_actions" {
  name = "yodex-github-actions"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = "repo:${var.github_owner}/${var.github_repo}:ref:refs/heads/${var.github_branch}"
          }
        }
      }
    ]
  })
}

data "aws_iam_policy_document" "s3_publish" {
  statement {
    sid     = "ListBucket"
    effect  = "Allow"
    actions = ["s3:ListBucket"]
    resources = [
      aws_s3_bucket.episodes.arn
    ]
  }

  statement {
    sid     = "PutGetObjects"
    effect  = "Allow"
    actions = ["s3:PutObject", "s3:GetObject"]
    resources = [
      "${aws_s3_bucket.episodes.arn}/*"
    ]
  }
}

# S3 permissions for the publish step.
resource "aws_iam_policy" "s3_publish" {
  name   = "yodex-s3-publish"
  policy = data.aws_iam_policy_document.s3_publish.json
}

resource "aws_iam_role_policy_attachment" "s3_publish" {
  role       = aws_iam_role.github_actions.name
  policy_arn = aws_iam_policy.s3_publish.arn
}

resource "aws_cloudfront_cache_policy" "latest_episode" {
  name        = "yodex-latest-episode"
  comment     = "Low TTL cache for latest episode"
  default_ttl = 300
  max_ttl     = 600
  min_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }

    headers_config {
      header_behavior = "none"
    }

    query_strings_config {
      query_string_behavior = "none"
    }

    enable_accept_encoding_brotli = true
    enable_accept_encoding_gzip   = true
  }
}

# DNS is managed in Cloudflare (not Route53), so ACM validation records must be
# created manually in Cloudflare using the terraform outputs.
resource "aws_acm_certificate" "podcast" {
  domain_name               = var.domain_name
  subject_alternative_names = ["www.${var.domain_name}"]
  validation_method         = "DNS"
  region                    = "us-east-1"
}

resource "aws_acm_certificate_validation" "podcast" {
  certificate_arn         = aws_acm_certificate.podcast.arn
  validation_record_fqdns = [for record in aws_acm_certificate.podcast.domain_validation_options : record.resource_record_name]
  region                  = "us-east-1"
}

resource "aws_cloudfront_function" "yoto_rewrite" {
  name    = "yodex-yoto-rewrite"
  runtime = "cloudfront-js-2.0"
  publish = true
  comment = "Rewrite /yoto/episode.mp3 to the latest episode"

  code = <<EOF
function handler(event) {
  var request = event.request;
  if (request.uri === "/yoto/episode.mp3") {
    request.uri = "/${local.full_prefix}/latest/episode.mp3";
  }
  return request;
}
EOF
}

resource "aws_cloudfront_distribution" "podcast" {
  enabled         = true
  is_ipv6_enabled = true
  comment         = "curiousworldpodcast.com distribution"

  aliases = [var.domain_name, "www.${var.domain_name}"]

  origin {
    domain_name = aws_s3_bucket.episodes.bucket_regional_domain_name
    origin_id   = "s3-episodes"
  }

  default_cache_behavior {
    target_origin_id       = "s3-episodes"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
    cache_policy_id        = aws_cloudfront_cache_policy.latest_episode.id

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.yoto_rewrite.arn
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.podcast.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}
