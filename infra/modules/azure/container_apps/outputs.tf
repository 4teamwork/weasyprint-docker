output "domain_name" {
  value = jsondecode(azapi_resource.container_app.output).properties.configuration.ingress.fqdn
}

output "id" {
  value = azapi_resource.container_app.id
}

output "identity" {
  value = azapi_resource.container_app.identity
}