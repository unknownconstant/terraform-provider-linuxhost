terraform {
  required_providers {
    linuxhost = {
      source = "example.com/util/linuxhost"
    }
  }
}

provider "linuxhost" {
  username    = "pi4"
  private_key = file("~/.ssh/id_rsa")
}

# data "linuxhost_test" "example" {

# }
//// sudo ip link add link eth0 name eth0.50 type vlan id 50
resource "linuxhost_network_interface" "ethx" {
  name             = "ethx"
  type             = "vlan"
  vlan_id          = 108
  parent_interface = "eth0"
  dhcp             = "dhclient"
}
# resource "linuxhost_network_interface_ip" "ethx1" {
#   interface_name = linuxhost_network_interface.ethx.name
#   ipv4           = "192.168.100.1/24"
# }

# resource "linuxhost_network_interface_ip" "ethx2" {
#   interface_name = linuxhost_network_interface.ethx.name
#   ipv4           = "192.168.100.2/24"
# }
# resource "linuxhost_network_interface_ip" "ethx3" {
#   interface_name = linuxhost_network_interface.ethx.name
#   ipv4           = "192.168.100.3/24"
# }

output "mac" {
  value = linuxhost_network_interface.ethx.mac
}
# resource "linuxhost_network_interface" "ethy" {
#   name = "ethy"
# }
# resource "linuxhost_network_interface_ip" "ethy1" {
#   interface_name = linuxhost_network_interface.ethy.name
#   ipv4           = "192.168.101.8/24"
# }

# resource "linuxhost_network_interface_ip" "ethy2" {
#   interface_name = linuxhost_network_interface.ethy.name
#   ipv4           = "192.168.101.2/24"
# }
# resource "linuxhost_network_interface_ip" "ethy3" {
#   interface_name = linuxhost_network_interface.ethy.name
#   ipv4           = "192.168.101.3/24"
# }

# output "mac2" {
#   value = linuxhost_network_interface.ethy.mac
# }
