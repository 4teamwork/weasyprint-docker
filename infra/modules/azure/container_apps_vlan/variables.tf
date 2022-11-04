variable "name" {
  description = "(Required) Specifies the name of the vlan"
  type        = string
}

variable "resource_group_name" {
  description = "(Required) Specifies the resource group name of the vlan"
  type        = string
}

variable "location" {
  description = "(Required) Specifies the location of the vlan"
  type        = string
}

variable "tags" {
  description = "(Optional) Specifies the tags of the vlan"
  default     = {}
}

variable "vlan_address_space" {
  description = "(Optional) Specifies the address space of the vlan"
  type		  = list(string)
  default     = ["10.0.0.0/16"]
}

variable "subnet_address_prefixes" {
  description = "(Optional) Specifies the address prefixes of the default subnet"
  type		  = list(string)
  default     = ["10.0.0.0/23"]
}

variable "subnet_name" {
  description = "(Optional) Specifies the name of the default subnet"
  type		  = string
  default     = "default"
}