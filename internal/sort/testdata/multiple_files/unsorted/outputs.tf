output "network_name" {
  value       = module.main.network_name
  description = "The name of the VPC being created"
}

output "network_self_link" {
  value       = module.main.network_self_link
  description = "The URI of the VPC being created"
}

output "subnets_names" {
  value       = module.main.subnets_names
  description = "The names of the subnets being created"
}

output "subnets_ips" {
  value       = module.main.subnets_ips
  description = "The IPs and CIDRs of the subnets being created"
}

output "subnets_self_links" {
  value       = module.main.subnets_self_links
  description = "The self-links of subnets being created"
}

output "subnets_regions" {
  value       = module.main.subnets_regions
  description = "The region where the subnets will be created"
}

output "subnets_private_access" {
  value       = module.main.subnets_private_access
  description = "Whether the subnets have access to Google API's without a public IP"
}

output "subnets_flow_logs" {
  value       = module.main.subnets_flow_logs
  description = "Whether the subnets have VPC flow logs enabled"
}

output "subnets_secondary_ranges" {
  value       = module.main.subnets_secondary_ranges
  description = "The secondary ranges associated with these subnets"
}

output "region1_router1" {
  value       = try(module.region1_router1[0], null)
  description = "Router 1 for Region 1"
}

output "region1_router2" {
  value       = try(module.region1_router2[0], null)
  description = "Router 2 for Region 1"
}

output "region2_router1" {
  value       = try(module.region2_router1[0], null)
  description = "Router 1 for Region 2"
}

output "region2_router2" {
  value       = try(module.region2_router2[0], null)
  description = "Router 2 for Region 2"
}
