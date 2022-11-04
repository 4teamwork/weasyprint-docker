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

variable "name" {
  description = "(Required) Friendly name of azure frontdoor instance."
  type        = string
}

variable "sku" {
  description = "(Required) Friendly name of azure frontdoor instance."
  type        = string
  default	  = "Standard_AzureFrontDoor"
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