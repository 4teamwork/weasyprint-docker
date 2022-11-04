variable "resource_group_id" {
  description = "(Required) The resource id of the resource group in which to create the resource. Changing this forces a new resource to be created."
  type        = string
}

variable "resource_group_name" {
  description = "(Required) The resource name of the resource group in which to create the resource. Should corrospond with provided resource group ID."
  type        = string
}


variable "name" {
  description = "(Required) Friendly name of azure frontdoor origin group and route."
  type        = string
}

variable "front_door_name" {
  description = "(Required) Name of frontdoor instance."
  type        = string
}

variable "origin_host" {
  description = "(Required) Host name of origin."
  type        = string
}

variable "endpoint_name" {
  description = "(Required) Name of the default endpoint for the frontdoor"
  type        = string
  default	  = "api"
}

variable "endpoint_domain_name" {
  description = "(Required) Name of the domain for the frontdoor endpoint"
  type        = string
}

variable "route_path" {
  description = "(Optional) Path to route to origin. Defaults to /{name}"
  type        = string
  nullable    = true
  default     = null
}

variable "origin_path" {
  description = "(Optional) Path on origin. Defaults to /"
  type        = string
  nullable    = true
  default     = "/"
}

variable "tags" {
  description = "(Optional) Specifies the tags of the log analytics workspace"
  type        = map(any)
  default     = {}
}