package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccROA(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	roa := func(name, extra string) string {
		return providerConfig(srv.URL) + fmt.Sprintf(`
resource "arin_roa" "test" {
  as_number = 64496
  name      = %q
  auto_link = true
  resources = [{
    start_address = "192.0.2.0"
    cidr_length   = 24%s
  }]
}
`, name, extra)
	}
	var firstHandle string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: roa("headquarters", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arin_roa.test", "roa_handle"),
					resource.TestCheckResourceAttr("arin_roa.test", "auto_renewed", "true"),
					resource.TestCheckResourceAttr("arin_roa.test", "resources.0.end_address", "192.0.2.255"),
					resource.TestCheckResourceAttr("arin_roa.test", "resources.0.ip_version", "4"),
					resource.TestCheckResourceAttr("arin_roa.test", "resources.0.auto_linked", "true"),
					resource.TestCheckNoResourceAttr("arin_roa.test", "resources.0.max_length"),
					func(s *terraform.State) error {
						firstHandle = s.RootModule().Resources["arin_roa.test"].Primary.Attributes["roa_handle"]
						if firstHandle == "" {
							return fmt.Errorf("empty roa_handle")
						}
						return nil
					},
				),
			},
			{
				Config: roa("renamed", "\n    max_length    = 25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_roa.test", "name", "renamed"),
					resource.TestCheckResourceAttr("arin_roa.test", "resources.0.max_length", "25"),
					func(s *terraform.State) error {
						h := s.RootModule().Resources["arin_roa.test"].Primary.Attributes["roa_handle"]
						if h == firstHandle {
							return fmt.Errorf("roa_handle unchanged after update")
						}
						return nil
					},
				),
			},
			{
				ResourceName: "arin_roa.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["arin_roa.test"].Primary.Attributes["roa_handle"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "roa_handle",
				// auto_link is write-only on the arin side
				// imports fall back to the schema default
				ImportStateVerifyIgnore: []string{"auto_link"},
			},
		},
	})
}
