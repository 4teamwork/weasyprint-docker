output "name" {
  value       = azapi_resource.managed_environment.name
  description = "Specifies the name of the managed environment."
}

output "id" {
  value       = azapi_resource.managed_environment.id
  description = "Specifies the resource id of the managed environment."
}