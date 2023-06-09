/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

variable "allow_all_egress_ranges" {
  default     = null
  description = "List of network ranges to which all egress traffic will be allowed"
}

variable "allow_all_ingress_ranges" {
  default     = null
  description = "List of network ranges from which all ingress traffic will be allowed"
}

variable "base_net_hub_project_id" {
  default     = ""
  description = "The base net hub project ID"
  type        = string
}

variable "bgp_asn_subnet" {
  description = "BGP ASN for Subnets cloud routers."
  type        = number
}

variable "default_region1" {
  description = "Default region 1 for subnets and Cloud Routers"
  type        = string
}

variable "default_region2" {
  description = "Default region 2 for subnets and Cloud Routers"
  type        = string
}

variable "dns_enable_inbound_forwarding" {
  default     = true
  description = "Toggle inbound query forwarding for VPC DNS."
  type        = bool
}

variable "dns_enable_logging" {
  default     = true
  description = "Toggle DNS logging for VPC DNS."
  type        = bool
}

variable "dns_hub_project_id" {
  description = "The DNS hub project ID"
  type        = string
}

variable "domain" {
  description = "The DNS name of peering managed zone, for instance 'example.com.'"
  type        = string
}

variable "environment_code" {
  description = "A short form of the folder level resources (environment) within the Google Cloud organization."
  type        = string
}

variable "firewall_enable_logging" {
  default     = true
  description = "Toggle firewall logging for VPC Firewalls."
  type        = bool
}

variable "mode" {
  default     = null
  description = "Network deployment mode, should be set to `hub` or `spoke` when `enable_hub_and_spoke` architecture chosen, keep as `null` otherwise."
  type        = string
}

variable "nat_bgp_asn" {
  default     = 64514
  description = "BGP ASN for first NAT cloud routes."
  type        = number
}

variable "nat_enabled" {
  default     = false
  description = "Toggle creation of NAT cloud router."
  type        = bool
}

variable "nat_num_addresses_region1" {
  default     = 2
  description = "Number of external IPs to reserve for first Cloud NAT."
  type        = number
}

variable "nat_num_addresses_region2" {
  default     = 2
  description = "Number of external IPs to reserve for second Cloud NAT."
  type        = number
}

variable "private_service_cidr" {
  default     = null
  description = "CIDR range for private service networking. Used for Cloud SQL and other managed services."
  type        = string
}

variable "private_service_connect_ip" {
  description = "Internal IP to be used as the private service connect endpoint."
  type        = string
}

variable "project_id" {
  description = "Project ID for Private Shared VPC."
  type        = string
}

variable "secondary_ranges" {
  default     = {}
  description = "Secondary ranges that will be used in some of the subnets"
  type        = map(list(object({ range_name = string, ip_cidr_range = string })))
}

variable "subnets" {
  default     = []
  description = "The list of subnets being created"
  type        = list(map(string))
}

variable "windows_activation_enabled" {
  default     = false
  description = "Enable Windows license activation for Windows workloads."
  type        = bool
}
