

provider "linuxhost" {
  host        = var.host
  username    = "tf"
  private_key = file("~/.ssh/id_rsa")
}

resource "linuxhost_user" "dummy" {
  username       = "dummy"
  home_directory = "/home/dummy"
  shell          = "/bin/bash"
}

resource "linuxhost_group" "foo" {
  name = "bar"
  gid  = "9010"
}