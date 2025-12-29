provider "aws" {
  region = var.region
}

provider "tls" {}

locals {
  full_prefix = trim(var.s3_prefix, "/")
}

resource "aws_s3_bucket" "episodes" {
  bucket = var.bucket_name

  tags = {
    Project = "yodex"
  }
}

resource "aws_s3_bucket_versioning" "episodes" {
  bucket = aws_s3_bucket.episodes.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "episodes" {
  bucket = aws_s3_bucket.episodes.id

  block_public_acls       = true
  ignore_public_acls      = true
  block_public_policy     = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "episodes_public" {
  bucket = aws_s3_bucket.episodes.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadEpisodes"
        Effect    = "Allow"
        Principal = "*"
        Action    = ["s3:GetObject"]
        Resource = [
          "${aws_s3_bucket.episodes.arn}/${local.full_prefix}/*"
        ]
      }
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.episodes]
}

# GitHub OIDC provider
resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = ["sts.amazonaws.com"]
  thumbprint_list = [
    data.tls_certificate.github.certificates[0].sha1_fingerprint
  ]
}

# Read the GitHub Actions OIDC cert chain for the thumbprint.
data "tls_certificate" "github" {
  url = "https://token.actions.githubusercontent.com"
}

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

resource "aws_iam_policy" "s3_publish" {
  name = "yodex-s3-publish"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ListBucket"
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.episodes.arn
        Condition = {
          StringLike = {
            "s3:prefix" = ["${local.full_prefix}/*"]
          }
        }
      },
      {
        Sid      = "PutGetObjects"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject"]
        Resource = "${aws_s3_bucket.episodes.arn}/${local.full_prefix}/*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "s3_publish" {
  role       = aws_iam_role.github_actions.name
  policy_arn = aws_iam_policy.s3_publish.arn
}
