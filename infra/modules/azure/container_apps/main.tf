terraform {
  required_version = ">= 1.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.3.0"
    }
    azapi = {
      source  = "Azure/azapi"
      version = "0.4.0"
    }
  }
  experiments = [module_variable_optional_attrs]
}

locals {
  module_tag = {
    "module" = basename(abspath(path.module))
  }
  tags = merge(var.tags, local.module_tag)
}

resource "azapi_resource" "daprComponents" {
  for_each  = {for component in var.dapr_components: component.name => component}

  name      = each.key
  parent_id = var.managed_environment_id
  type      = "Microsoft.App/managedEnvironments/daprComponents@2022-03-01"

  body = jsonencode({
    properties = {
      componentType   = each.value.componentType
      version         = each.value.version
      ignoreErrors    = each.value.ignoreErrors
      initTimeout     = each.value.initTimeout
      secrets         = each.value.secrets
      metadata        = each.value.metadata
      scopes          = each.value.scopes
    }
  })
}

#ref
# https://raw.githubusercontent.com/Azure/azure-resource-manager-schemas/68af7da6820cc91660904b34813aeee606c400f1/schemas/2022-03-01/Microsoft.App.json

resource "azapi_resource" "container_app" {
  # for_each  = {for app in var.container_apps: app.name => app}

  name      = var.container_app.name
  location  = var.location
  parent_id = var.resource_group_id
  type      = "Microsoft.App/containerApps@2022-03-01"
  tags      = var.tags
  
  body = jsonencode({
    properties = {

      managedEnvironmentId  = var.managed_environment_id
      configuration         = {
        registries = [for reg in var.registries : {
          server = reg.login_server
          username = reg.admin_username
          passwordSecretRef = "acr-pw-${replace(reg.login_server,".","-")}"
        }]
        secrets = concat((var.container_app.configuration.secrets == null ? [] : var.container_app.configuration.secrets), [for reg in var.registries : {
          name = "acr-pw-${replace(reg.login_server,".","-")}"
          value = reg.admin_password
        }])
        ingress             = try(var.container_app.configuration.ingress, null)
        dapr                = try(var.container_app.configuration.dapr, null)
      }
      template              = var.container_app.template
      
    }
  })
  
  response_export_values = ["properties.configuration.ingress.fqdn"]

}