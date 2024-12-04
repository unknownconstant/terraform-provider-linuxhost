terraform {
  required_providers {
    linuxhost = {
      source = "example.com/util/linuxhost"
    }
  }
}

variable "host" {
  type = string
}

provider "linuxhost" {
  host        = var.host
  username    = "tf"
  private_key = file("~/.ssh/id_rsa")
}

# resource "linuxhost_ca_certificate" "self" {
#   name   = "test"
#   source = "cert.pem"
# }
