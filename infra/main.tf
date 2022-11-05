terraform {
  required_version = ">= 1.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.3.0"
    }

    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.15.0"
    }

    azapi = {
      source  = "Azure/azapi"
      version = "0.4.0"
    }
  }
  experiments = [module_variable_optional_attrs]

  backend "azurerm" {
        resource_group_name  = "BCC-Platform"
        storage_account_name = "bccplatformtfstate"
        container_name       = "pdf-service"
        key                  = "pdf-service.terraform.tfstate"
        subscription_id      = "a77a3461-9212-44cf-bc6a-11c6281797e9"
        tenant_id            = "8572f54e-d0a8-4ea4-a28e-557c63698a4a"
  }

}

locals {
    azure_tenant_id             = var.azure_tenant_id
    azure_subscription_id       = var.azure_subscription_id
    azure_platform_subscription_id = var.azure_platform_subscription_id
    location                    = var.location
    resource_group              = "${var.resource_prefix}-${var.environment}"
    resource_prefix             = "${var.resource_prefix}"
    tags                        = {}
}

provider "azuread" {
  tenant_id = var.azure_tenant_id
  features {}
}

provider "azurerm" {
  alias             = "main"
  subscription_id   = local.azure_subscription_id
  tenant_id         = local.azure_tenant_id
  skip_provider_registration = true
  features {}
}

provider "azurerm" {
  alias             = "platform"
  subscription_id   = local.azure_platform_subscription_id
  tenant_id         = local.azure_tenant_id
  skip_provider_registration = true
  features {}
}


provider "azapi" {
  subscription_id   = local.azure_subscription_id
  tenant_id         = local.azure_tenant_id 
  skip_provider_registration = true
}


# Get Resource Group
data "azurerm_resource_group" "rg" {
  provider = azurerm.main
  name     = local.resource_group
}

# Get Container Registry
data "azurerm_container_registry" "acr" {
  provider            = azurerm.platform
  name                = "bccplatform"
  resource_group_name = "BCC-Platform"
}

# Analytics Workspace
module "log_analytics_workspace" {
  source                           = "./modules/azure/log_analytics"
  name                             = "${local.resource_prefix}-logs"
  location                         = local.location
  resource_group_name              = data.azurerm_resource_group.rg.name
  tags                             = local.tags
  providers = {
    azurerm = azurerm.main
  }
}

# Application Insights
module "application_insights" {
  source                           = "./modules/azure/application_insights"
  name                             = "${local.resource_prefix}-env-insights"
  location                         = local.location
  resource_group_name              = data.azurerm_resource_group.rg.name
  tags                             = local.tags
  application_type                 = "web"
  workspace_id                     = module.log_analytics_workspace.id
  providers = {
    azurerm = azurerm.main
  }
}

# VLAN for Container Environment
module "container_apps_vlan" {
  source                           = "./modules/azure/container_apps_vlan"
  name                             = "${local.resource_prefix}-vlan"
  location                         = local.location
  resource_group_name              = data.azurerm_resource_group.rg.name
  tags                             = local.tags

  depends_on = [
    data.azurerm_resource_group.rg
  ]

  providers = {
    azurerm = azurerm.main
  }
}


# Container Environment
module "container_apps_env"  {
  source                           = "./modules/azure/container_apps_env"
  managed_environment_name         = "${local.resource_prefix}-env"
  location                         = local.location
  resource_group_id                = data.azurerm_resource_group.rg.id
  tags                             = local.tags
  instrumentation_key              = module.application_insights.instrumentation_key
  workspace_id                     = module.log_analytics_workspace.workspace_id
  primary_shared_key               = module.log_analytics_workspace.primary_shared_key
  vlan_subnet_id                   = module.container_apps_vlan.subnet_id

  providers = {
    azurerm = azurerm.main
  }
}


#ref:
# https://github.com/Azure/azure-resource-manager-schemas/blob/68af7da6820cc91660904b34813aeee606c400f1/schemas/2022-03-01/Microsoft.App.json

# API Container App
module "api_container_app" {
  source                           = "./modules/azure/container_apps"
  managed_environment_id           = module.container_apps_env.id
  location                         = local.location
  resource_group_id                = data.azurerm_resource_group.rg.id
  tags                             = local.tags
  registries = [{
    admin_password = data.azurerm_container_registry.acr.admin_password
    admin_username = data.azurerm_container_registry.acr.admin_username
    login_server = data.azurerm_container_registry.acr.login_server
  }]
  container_app                   = {
    name              = "${local.resource_prefix}"
    configuration      = {
      ingress          = {
        external       = true
        targetPort     = 5130
      }
      dapr             = {
        enabled        = true
        appId          = "${local.resource_prefix}"
        appProtocol    = "http"
        appPort        = 5130
      }
      secrets          = []
      # customDomains  = [
      #   {
      #     bindingType   = "SniEnabled",
      #     certificateId = "",
      #     name          = module.api_container_app.domain_name
      #   }
      # ]
    }
    template          = {
      containers      = [{
        image         = "hello-world:latest" //"bccplatform.azurecr.io/bcc-code-run-prod-api:latest"
        name          = "${local.resource_prefix}"
        env           = [{
            name        = "APP_PORT"
            value       = 8080
          },
          {
            name        = "ENVIRONMENT_NAME"
            value       = terraform.workspace
          }
        ]
        resources     = {
          cpu         = 0.5
          memory      = "1Gi"
        }
      },
      {
        image         = "hello-world:latest" //"bccplatform.azurecr.io/bcc-code-run-prod-api:latest"
        name          = "${local.resource_prefix}-proxy"
        env           = [{
            name        = "APP_PORT"
            value       = 5130
          },
          {
            name        = "ASPNETCORE_URLS"
            value       = "http://+:5130"
          },
          {
            name        = "ENVIRONMENT_NAME"
            value       = terraform.workspace
          }
        ]
        resources     = {
          cpu         = 0.5
          memory      = "1Gi"
        }
      }]
      scale           = {
        minReplicas   = 0
        maxReplicas   = 10
      }
    }
  }
  providers = {
    azurerm = azurerm.main
  }
}

# # # Add gateway

# module "gateway" {
#   source                = "./modules/azure/front_door"
#   name                  = "${local.resource_prefix}-gateway"
#   location              = local.location
#   tags                  = local.tags
#   endpoint_domain_name  = var.endpoint_domain_name
#   endpoint_name         = "default"
#   resource_group_id     = data.azurerm_resource_group.rg.id
#   providers = {
#     azurerm = azurerm.main
#   }
# }

# module "api_route" {
#   source                = "./modules/azure/front_door_route"
#   name                  = "${local.resource_prefix}-svc-route"
#   front_door_name       = "${local.resource_prefix}-gateway"
#   origin_host           = module.api_container_app.domain_name
#   route_path            = "/*"
#   origin_path           = "/" 
#   endpoint_name         = "default"
#   endpoint_domain_name  = var.endpoint_domain_name
#   resource_group_id     = data.azurerm_resource_group.rg.id
#   resource_group_name   = data.azurerm_resource_group.rg.name 
#   depends_on = [
#     module.gateway
#   ]
#   providers = {
#     azurerm = azurerm.main
#   }
# }