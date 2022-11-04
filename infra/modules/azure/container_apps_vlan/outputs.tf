output "id" {
  value = azurerm_virtual_network.vlan.id
  description = "Specifies the id of the virtual network"
}

output "location" {
  value = azurerm_virtual_network.vlan.location
  description = "Specifies the location of the virtual network"
}

output "network_security_group_id" {
  value = azurerm_network_security_group.nsg.id
  description = "Specifies the resource id of the network security group"
}

output "subnet_id" {
  value = azurerm_subnet.default.id
  description = "Specifies the resource id of the default subnet"
}

