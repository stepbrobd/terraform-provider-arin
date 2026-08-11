package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccIRRSets(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	cfg := func(members string) string {
		return providerConfig(srv.URL) + fmt.Sprintf(`
resource "arin_irr_as_set" "test" {
  name         = "AS64496:AS-EXAMPLE"
  descriptions = ["example"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
  members      = [%s]
  mbrs_by_ref  = ["MNT-TESTORG"]
}

resource "arin_irr_route_set" "test" {
  name         = "RS-EXAMPLE"
  descriptions = ["example"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
  mp_members   = ["192.0.2.0/24", "2001:db8::/32"]
}
`, members)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: cfg(`"AS64496"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_as_set.test", "members.#", "1"),
					resource.TestCheckResourceAttr("arin_irr_route_set.test", "mp_members.#", "2"),
					resource.TestCheckResourceAttrSet("arin_irr_as_set.test", "created"),
				),
			},
			{
				Config: cfg(`"AS64496", "AS64497:AS-MORE"`),
				Check:  resource.TestCheckResourceAttr("arin_irr_as_set.test", "members.#", "2"),
			},
			{
				ResourceName:                         "arin_irr_as_set.test",
				ImportState:                          true,
				ImportStateId:                        "AS64496:AS-EXAMPLE",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				ResourceName:                         "arin_irr_route_set.test",
				ImportState:                          true,
				ImportStateId:                        "RS-EXAMPLE",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}
