package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccIRRAutNum(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	autnum := func(name string, extra string) string {
		return providerConfig(srv.URL) + fmt.Sprintf(`
resource "arin_irr_aut_num" "test" {
  as_number    = 64496
  as_name      = %q
  descriptions = ["example"]%s
}
`, name, extra)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: autnum("EXAMPLE", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "as_name", "EXAMPLE"),
					resource.TestCheckResourceAttrSet("arin_irr_aut_num.test", "created"),
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "org_handle", "TESTORG"),
				),
			},
			{
				Config: autnum("RENAMED", "\n  mp_imports   = [\"afi any from AS64497 accept ANY\"]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "as_name", "RENAMED"),
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "mp_imports.0", "afi any from AS64497 accept ANY"),
				),
			},
			{
				ResourceName:                         "arin_irr_aut_num.test",
				ImportState:                          true,
				ImportStateId:                        "64496",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "as_number",
			},
		},
	})
}
