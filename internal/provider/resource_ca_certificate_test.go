package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCertificate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linuxhost_ca_certificate.self", "name", "test"),
				),
			},
		},
	})
}

func testAccCertificateResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
provider "linuxhost" {
  username    = "tf"
  private_key = file("~/.ssh/id_rsa")
}
resource "linuxhost_ca_certificate" "self" {
  name   = %[1]q
  source = "../../examples/ca_cert/cert.pem"
}
`, configurableAttribute)
}
