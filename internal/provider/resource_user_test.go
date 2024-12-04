package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExampleResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linuxhost_user.dummy", "username", "one"),
				),
			},
		},
	})
}

func testAccExampleResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
provider "linuxhost" {
  username    = "tf"
  private_key = file("~/.ssh/id_rsa")
}
resource "linuxhost_user" "dummy" {
  username       = %[1]q
  home_directory = "/home/dummy"
  shell          = "/bin/bash"
}
`, configurableAttribute)
}
