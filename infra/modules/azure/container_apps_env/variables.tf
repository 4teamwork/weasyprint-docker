
variable "managed_environment_name" {
  description = "(Required) Specifies the name of the managed environment."
  type        = string
}

variable "resource_group_id" {
  description = "(Required) The resource id of the resource group in which to create the resource. Changing this forces a new resource to be created."
  type        = string
}

variable "tags" {
  description = "(Optional) Specifies the tags of the log analytics workspace"
  type        = map(any)
  default     = {}
}

variable "location" {
  description = "(Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created."
  type        = string
}

variable "instrumentation_key" {
  description = "(Optional) Specifies the instrumentation key of the application insights resource."
  type        = string
}

variable "workspace_id" {
  description = "(Optional) Specifies workspace id of the log analytics workspace."
  type        = string
}

variable "primary_shared_key" {
  description = "(Optional) Specifies the workspace key of the log analytics workspace."
  type        = string
}

variable "vlan_subnet_id" {
  description = "(Required) Specifies the id of the vlan subnet for the managed environment."
  type        = string
}


