variable "azure_subscription_id" {
  type      = string
  default   = "7e26d6b5-d4a4-4eea-bfd8-ddac86d350b4" #bcc-core-prod
}

variable "azure_platform_subscription_id" {
  type      = string
  default   = "a77a3461-9212-44cf-bc6a-11c6281797e9" #BCC Pay-as-you-go
}

variable "azure_tenant_id" {
  type      = string
  default   = "8572f54e-d0a8-4ea4-a28e-557c63698a4a"
}

variable "environment" {
  type = string
  default = "prod"
}

variable "resource_prefix" {
  type    = string
  default = "pdf-service"
}

variable "endpoint_domain_name" {
  type    = string
  default = "pdf-service.bcc.no"
}

variable "location" {
  type      = string
  default   = "westeurope"
}