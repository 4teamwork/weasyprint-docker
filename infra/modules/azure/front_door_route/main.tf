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
  route_path = var.route_path == null ? "/${var.name}/*" : var.route_path
  origin_path = var.origin_path == null ? "/" : var.origin_path
}


data "azapi_resource" "frontdoor_profile" {
  name      = var.front_door_name
  parent_id = var.resource_group_id
  type      = "Microsoft.Cdn/profiles@2021-06-01"
}

data "azapi_resource" "frontdoor_endpoint" {
  name      = var.endpoint_name
  parent_id = data.azapi_resource.frontdoor_profile.id
  type      = "Microsoft.Cdn/profiles/afdEndpoints@2021-06-01"
}

data "azapi_resource" "endpoint_domain" {
  name      = replace(var.endpoint_domain_name,".","-")
  parent_id = data.azapi_resource.frontdoor_profile.id
  type      = "Microsoft.Cdn/profiles/customdomains@2021-06-01"
}

data "azurerm_client_config" "current" {}

resource "azapi_resource" "origin_group" {

  name      = "${var.name}-origin-group"
  parent_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/resourceGroups/${var.resource_group_name}/providers/Microsoft.Cdn/profiles/${var.front_door_name}" 
  #data.azapi_resource.frontdoor_profile.id
  type      = "Microsoft.Cdn/profiles/origingroups@2021-06-01"

  body = jsonencode({
    properties = {
        healthProbeSettings = {
            probePath = "/"
            probeIntervalInSeconds = 255
            probeProtocol = "Https"
            probeRequestType = "HEAD"
        }
        sessionAffinityState = "Enabled"
        loadBalancingSettings = {
            additionalLatencyInMilliseconds = 50
            successfulSamplesRequired = 3
            sampleSize = 4
        }
    }
  })
  
  depends_on = [
    data.azapi_resource.frontdoor_profile
  ]

}

resource "azapi_resource" "origin" {

  name      = "${var.name}"
  parent_id = azapi_resource.origin_group.id
  type      = "Microsoft.Cdn/profiles/origingroups/origins@2021-06-01"

  body = jsonencode({
    properties = {
        # azureOrigin = {
        #     id = azapi_resource.origin_group.id
        # }
       hostName = var.origin_host
       httpPort = 80
       httpsPort = 443
       originHostHeader = var.origin_host
       enabledState = "Enabled"
       priority = 1
       weight = 1000
    }
  })
  
  depends_on = [
    azapi_resource.origin_group
  ]

}

# DOESN'T CURRENLTY WORK - Causes an error, presumably because this resource can't be looked up using the CLi
# resource "azapi_resource" "origin_route" {

#   name      = "${var.name}"
#   parent_id = "${data.azapi_resource.frontdoor_endpoint.id}"
#   type      = "Microsoft.Cdn/profiles/afdendpoints/routes@2021-06-01"

#   body = jsonencode({
#     properties = {
#       originPath = "/"

#         # customDomains = [
#         #     {
#         #         id = data.azapi_resource.endpoint_domain.id
#         #     }
#         # ]
#         originGroup = {
#             id = azapi_resource.origin_group.id
#         }
#         supportedProtocols = [
#           "Http",
#           "Https"
#         ]
#         patternsToMatch = [
#             local.route_path
#         ]
#         forwardingProtocol = "MatchRequest"
#         linkToDefaultDomain = "Enabled"
#         httpsRedirect = "Enabled"
#         enabledState = "Enabled"
#     }
#   })
  
#   # depends_on = [
#   #   azapi_resource.origin_group,
#   # ]

# }

resource "random_id" "id" {
	  byte_length = 8
}

resource "azurerm_resource_group_template_deployment" "origin_route" {
  name                = "${var.name}_deployment_${random_id.id.hex}"
  resource_group_name = "${var.resource_group_name}"
  deployment_mode     = "Incremental"
  parameters_content = jsonencode({
    "frontdoor_name" = {
      value = var.front_door_name
    }
    "endpoint_name" = {
      value = var.endpoint_name
    }
    "route_name" = {
      value = var.name
    }
    "route_path" = {
      value = local.route_path
    }
    "origin_path" = {
      value = local.origin_path
    }
    "origin_group_name" = {
      value = azapi_resource.origin_group.name
    }
    "endpoint_domain_name" = {
      value = replace(var.endpoint_domain_name,".","-")
    }
  })
  template_content = <<TEMPLATE
{
    "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "frontdoor_name": {
            "type": "string"
        },
        "endpoint_name": {
            "type": "string"
        },
        "route_name": {
            "type": "string"
        },
        "route_path": {
            "type": "string"
        },
        "origin_path": {
            "type": "string"
        },
        "origin_group_name": {
            "type": "string"
        },
        "endpoint_domain_name": {
            "type": "string"
        }
    },
    "variables": {},
    "resources": [
        {
            "type": "Microsoft.Cdn/profiles/afdendpoints/routes",
            "apiVersion": "2021-06-01",
            "name": "[concat(parameters('frontdoor_name'), '/', parameters('endpoint_name'), '/', parameters('route_name'))]",
            "properties": {
                "customDomains": [{
                        "id": "[resourceId('Microsoft.Cdn/profiles/customdomains', parameters('frontdoor_name'), parameters('endpoint_domain_name'))]"
                    }],
                "originGroup": {
                    "id": "[resourceId('Microsoft.Cdn/profiles/origingroups', parameters('frontdoor_name'), parameters('origin_group_name'))]"
                },
                "originPath": "[parameters('origin_path')]",
                "ruleSets": [],
                "supportedProtocols": [
                    "Http",
                    "Https"
                ],
                "patternsToMatch": [
                    "[parameters('route_path')]"
                ],
                "forwardingProtocol": "MatchRequest",
                "linkToDefaultDomain": "Enabled",
                "httpsRedirect": "Enabled",
                "enabledState": "Enabled"
            }
        }
    ],
    "outputs": {}
}
TEMPLATE

 depends_on = [
   azapi_resource.origin
 ]

}