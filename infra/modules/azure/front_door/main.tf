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

resource "azapi_resource" "frontdoor_profile" {

  name      = var.name
  location  = "global"
  parent_id = var.resource_group_id
  type      = "Microsoft.Cdn/profiles@2021-06-01"
  tags      = local.tags

  body = jsonencode({
    sku = {
      name = var.sku
    }
  })

  lifecycle {
    ignore_changes = [
        tags
    ]
  }
}

resource "azapi_resource" "frontdoor_endpoint" {

  name      = var.endpoint_name
  location  = "global"
  parent_id = azapi_resource.frontdoor_profile.id
  type      = "Microsoft.Cdn/profiles/afdEndpoints@2021-06-01"
  tags      = local.tags

  body = jsonencode({
    properties = {
      enabledState = "Enabled"
    }
  })

  lifecycle {
    ignore_changes = [
        tags
    ]
  }
}

resource "azapi_resource" "frontdoor_domain" {
  name = "${replace(var.endpoint_domain_name,".","-")}"
  parent_id = azapi_resource.frontdoor_profile.id
  type      = "Microsoft.Cdn/profiles/customdomains@2021-06-01"

  body = jsonencode({
    properties = {
      hostName = var.endpoint_domain_name,
      tlsSettings = {
        certificateType = "ManagedCertificate"
        minimumTlsVersion = "TLS12"
      }
    }
  })

  lifecycle {
    ignore_changes = [
        tags
    ]
  }
}