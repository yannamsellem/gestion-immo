output "app_r2_bucket_name" {
  description = "Name of the Cloudflare R2 bucket for application documents and photos"
  value       = cloudflare_r2_bucket.app_files.name
}

output "backup_r2_bucket_name" {
  description = "Name of the Cloudflare R2 bucket for PostgreSQL backups"
  value       = cloudflare_r2_bucket.pg_backups.name
}

output "tunnel_id" {
  description = "Cloudflare Zero Trust Tunnel ID"
  value       = cloudflare_zero_trust_tunnel_cloudflared.vps_tunnel.id
}

output "tunnel_token" {
  description = "Cloudflare Tunnel token for the cloudflared container on the VPS"
  value       = cloudflare_zero_trust_tunnel_cloudflared.vps_tunnel.tunnel_token
  sensitive   = true
}

output "pages_subdomain" {
  description = "Cloudflare Pages subdomain for the SvelteKit SSR frontend"
  value       = cloudflare_pages_project.web_frontend.subdomain
}

output "api_base_url" {
  description = "Public HTTPS API URL routed through the Cloudflare Tunnel"
  value       = "https://api.${var.domain_name}"
}

output "mcp_endpoint_url" {
  description = "Public HTTPS MCP Streamable HTTP endpoint for Antigravity"
  value       = "https://mcp.${var.domain_name}/mcp"
}
