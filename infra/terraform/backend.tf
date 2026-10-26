terraform {
  required_version = ">= 1.17.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.40"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Remote state stored in a dedicated Cloudflare R2 bucket (S3-compatible API).
  # Credentials and endpoint are supplied via environment variables or -backend-config at `terraform init`.
  backend "s3" {
    bucket                      = "gestion-immo-tfstate"
    key                         = "prod/terraform.tfstate"
    region                      = "auto"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true
    use_path_style              = true
    access_key                  = "<YOUR_R2_ACCESS_KEY>"
    secret_key                  = "<YOUR_R2_ACCESS_SECRET>"
    endpoints                   = { s3 = "https://${var.cloudflare_account_id}.r2.cloudflarestorage.com" }
  }
}
