provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Application R2 bucket (leases, inspection photos, templates, GLI archives, Liasse PDFs)
resource "cloudflare_r2_bucket" "app_files" {
  account_id = var.cloudflare_account_id
  name       = var.app_r2_bucket_name
  location   = "WEUR"
}

# Daily PostgreSQL backup R2 bucket
resource "cloudflare_r2_bucket" "pg_backups" {
  account_id = var.cloudflare_account_id
  name       = var.backup_r2_bucket_name
  location   = "WEUR"
}

# Random secret for the Cloudflare Zero Trust Tunnel
resource "random_id" "tunnel_secret" {
  byte_length = 32
}

# Cloudflare Tunnel linking Cloudflare Pages & Antigravity MCP clients to the Hostinger VPS Go monolith
resource "cloudflare_zero_trust_tunnel_cloudflared" "vps_tunnel" {
  account_id = var.cloudflare_account_id
  name       = "gestion-immo-vps-tunnel"
  secret     = random_id.tunnel_secret.b64_std
  config_src = "cloudflare"
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "vps_tunnel_config" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.vps_tunnel.id

  config {
    ingress_rule {
      hostname = "api.${var.domain_name}"
      service  = "http://gestion-immo:8080"
    }
    ingress_rule {
      hostname = "mcp.${var.domain_name}"
      service  = "http://gestion-immo:8080"
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}

# DNS CNAME records routing api.<domain> and mcp.<domain> through the Cloudflare Tunnel
resource "cloudflare_record" "api_dns" {
  zone_id = var.cloudflare_zone_id
  name    = "api"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.vps_tunnel.id}.cfargotunnel.com"
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "mcp_dns" {
  zone_id = var.cloudflare_zone_id
  name    = "mcp"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.vps_tunnel.id}.cfargotunnel.com"
  type    = "CNAME"
  proxied = true
}

# Cloudflare Pages project hosting the SvelteKit SSR frontend (shadcn-svelte)
resource "cloudflare_pages_project" "web_frontend" {
  account_id        = var.cloudflare_account_id
  name              = var.pages_project_name
  production_branch = "main"

  build_config {
    build_command   = "npm run build"
    destination_dir = ".svelte-kit/cloudflare"
    root_dir        = "web"
  }

  deployment_configs {
    production {
      environment_variables = {
        PUBLIC_API_BASE_URL = "https://api.${var.domain_name}"
      }
      compatibility_date  = "2026-10-01"
      compatibility_flags = ["nodejs_compat"]
    }
  }
}
