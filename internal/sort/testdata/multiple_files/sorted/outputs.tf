output "network_name" {
  description = "The name of the VPC being created"
  value       = module.main.network_name
}

output "network_self_link" {
  description = "The URI of the VPC being created"
  value       = module.main.network_self_link
}

output "region1_router1" {
  description = "Router 1 for Region 1"
  value       = try(module.region1_router1[0], null)
}

output "region1_router2" {
  description = "Router 2 for Region 1"
  value       = try(module.region1_router2[0], null)
}

output "region2_router1" {
  description = "Router 1 for Region 2"
  value       = try(module.region2_router1[0], null)
}

output "region2_router2" {
  description = "Router 2 for Region 2"
  value       = try(module.region2_router2[0], null)
}

output "subnets_flow_logs" {
  description = "Whether the subnets have VPC flow logs enabled"
  value       = module.main.subnets_flow_logs
}

output "subnets_ips" {
  description = "The IPs and CIDRs of the subnets being created"
  value       = module.main.subnets_ips
}

output "subnets_names" {
  description = "The names of the subnets being created"
  value       = module.main.subnets_names
}

output "subnets_private_access" {
  description = "Whether the subnets have access to Google API's without a public IP"
  value       = module.main.subnets_private_access
}

output "subnets_regions" {
  description = "The region where the subnets will be created"
  value       = module.main.subnets_regions
}

output "subnets_secondary_ranges" {
  description = "The secondary ranges associated with these subnets"
  value       = module.main.subnets_secondary_ranges
}

output "subnets_self_links" {
  description = "The self-links of subnets being created"
  value       = module.main.subnets_self_links
}
