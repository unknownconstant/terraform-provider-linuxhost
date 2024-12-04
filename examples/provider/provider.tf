provider "linuxhost" {
  host        = var.host
  username    = "terraform"
  private_key = file("~/.ssh/id_rsa")
}