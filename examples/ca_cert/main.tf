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
variable "username" {
  type = string
}

provider "linuxhost" {
  host        = var.host
  username    = var.username
  private_key = file("~/.ssh/id_rsa")
}

resource "linuxhost_ca_certificate" "self" {
  name   = "test"
  source = "cert.pem"
  # certificate= file("cert.pem")
}

output "serialnumber" {
  value = linuxhost_ca_certificate.self.serial_number
}
output "certificate" {
  value = linuxhost_ca_certificate.self.certificate
}
