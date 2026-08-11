package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccIRRData(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	cfg := providerConfig(srv.URL) + `
resource "arin_irr_route" "a" {
  prefix       = "192.0.2.0/24"
  origin_as    = 64496
  descriptions = ["r"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
}

resource "arin_irr_aut_num" "a" {
  as_number    = 64496
  as_name      = "EXAMPLE"
  descriptions = ["a"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
}

resource "arin_irr_as_set" "a" {
  name         = "AS64496:AS-EXAMPLE"
  descriptions = ["s"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
}

resource "arin_irr_route_set" "a" {
  name         = "RS-EXAMPLE"
  descriptions = ["s"]
  admin_pocs   = ["EXA-ARIN"]
  tech_pocs    = ["EXT-ARIN"]
}

data "arin_irr_routes" "all" {
  depends_on = [arin_irr_route.a]
}

data "arin_irr_aut_nums" "all" {
  depends_on = [arin_irr_aut_num.a]
}

data "arin_irr_as_sets" "all" {
  depends_on = [arin_irr_as_set.a]
}

data "arin_irr_route_sets" "all" {
  depends_on = [arin_irr_route_set.a]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.arin_irr_routes.all", "routes.#", "1"),
					resource.TestCheckResourceAttr("data.arin_irr_routes.all", "routes.0.prefix", "192.0.2.0/24"),
					resource.TestCheckResourceAttr("data.arin_irr_aut_nums.all", "aut_nums.#", "1"),
					resource.TestCheckResourceAttr("data.arin_irr_aut_nums.all", "aut_nums.0.as_number", "64496"),
					resource.TestCheckResourceAttr("data.arin_irr_as_sets.all", "as_sets.#", "1"),
					resource.TestCheckResourceAttr("data.arin_irr_route_sets.all", "route_sets.#", "1"),
				),
			},
		},
	})
}
