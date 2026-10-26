variable "cloudflare_api_token" {
  description = "Cloudflare API token with permissions for R2, Zero Trust Tunnels, Pages, and DNS"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare Account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare Zone ID for the primary domain"
  type        = string
}

variable "domain_name" {
  description = "Primary domain name for Gestion Immo"
  type        = string
}

variable "app_r2_bucket_name" {
  description = "Cloudflare R2 bucket name for application files, inspection photos, templates, and generated PDFs"
  type        = string
  default     = "gestion-immo-files"
}

variable "backup_r2_bucket_name" {
  description = "Cloudflare R2 bucket name for encrypted daily PostgreSQL backups"
  type        = string
  default     = "gestion-immo-backups"
}

variable "pages_project_name" {
  description = "Cloudflare Pages project name for the SvelteKit SSR frontend"
  type        = string
  default     = "gestion-immo-web"
}
