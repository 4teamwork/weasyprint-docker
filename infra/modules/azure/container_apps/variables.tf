
variable "managed_environment_id" {
  description = "(Required) Specifies the id of the managed environment."
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

variable "container_app" {
  description = "Specifies the container apps in the managed environment."
  type = object({
    name                = string
    configuration       = object({
      ingress           = optional(object({
        external        = optional(bool)
        targetPort      = optional(number)
      }))
      dapr              = optional(object({
        enabled         = optional(bool)
        appId           = optional(string)
        appProtocol     = optional(string)
        appPort         = optional(number)
      }))
      secrets            = optional(list(object({
        name             = string
        value            = string
      })))
    })
    template           = object({
      containers       = list(object({
        image          = string
        name           = string
        env            = optional(list(object({
          name         = string
          value        = optional(string)
          secretRef    = optional(string)
        })))
        resources      = optional(object({
          cpu          = optional(number)
          memory       = optional(string)
        }))
      }))
      scale            = optional(object({
        minReplicas    = optional(number)
        maxReplicas    = optional(number)
      }))
    })
  })
}

variable "dapr_components" {
  description = "Specifies the dapr components in the managed environment."
  type = list(object({
    name           = string
    componentType  = string
    version        = string
    ignoreErrors   = optional(bool)
    initTimeout    = string
    secrets        = optional(list(object({
      name         = string
      value        = any
    })))
    metadata       = optional(list(object({
      name         = string
      value        = optional(any)
      secretRef    = optional(any)
    })))
    scopes         = optional(list(string))
  }))
  default          = []
}