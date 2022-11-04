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

resource "azapi_resource" "managed_environment" {
  name      = var.managed_environment_name
  location  = var.location
  parent_id = var.resource_group_id
  type      = "Microsoft.App/managedEnvironments@2022-03-01"
  tags      = local.tags
  
  body = jsonencode({
    properties = {
      daprAIInstrumentationKey = var.instrumentation_key
      appLogsConfiguration = {
        destination = "log-analytics"
        logAnalyticsConfiguration = {
          customerId = var.workspace_id
          sharedKey  = var.primary_shared_key
        }
      }
      vnetConfiguration = {
        infrastructureSubnetId = var.vlan_subnet_id
        internal = false
      }
      zoneRedundant = true
    }
  })

  lifecycle {
    ignore_changes = [
        tags
    ]
  }
}
