provider "aws" {
  region = var.region
}

provider "tls" {}

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
