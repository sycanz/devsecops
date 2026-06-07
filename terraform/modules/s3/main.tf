resource "aws_s3_bucket" "devsecops-tfstate" {
  bucket = "devsecops-tfstate-sycanz"

  tags = {
    Name        = "devsecops bucket"
    Environment = "dev"
  }
}

resource "aws_s3_bucket_versioning" "tfstate" {
  bucket = aws_s3_bucket.devsecops-tfstate.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_kms_key" "tfstate" {
  description             = "S3 state bucket encryption key"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
  bucket = aws_s3_bucket.devsecops-tfstate.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.tfstate.arn
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tfstate" {
  bucket = aws_s3_bucket.devsecops-tfstate.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "tfstate" {
  bucket = aws_s3_bucket.devsecops-tfstate.id

  rule {
    id = "DeleteOldLockFiles"

    status = "Enabled"

    filter {
      prefix = "state/"
    }
    
    expiration {
      days = 7
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }
}
