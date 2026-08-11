package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin"
	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccIRRRoute(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	route := func(descr, remarks string) string {
		return providerConfig(srv.URL) + fmt.Sprintf(`
resource "arin_irr_route" "test" {
  prefix       = "192.0.2.0/24"
  origin_as    = 64496
  descriptions = [%q]
  remarks      = %s
}
`, descr, remarks)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: route("first", `["r1"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// state stays canonical even though the fake serves padded
					resource.TestCheckResourceAttr("arin_irr_route.test", "prefix", "192.0.2.0/24"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "ip_version", "4"),
					resource.TestCheckResourceAttrSet("arin_irr_route.test", "net_handle"),
					resource.TestCheckResourceAttrSet("arin_irr_route.test", "created"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "org_handle", "TESTORG"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "descriptions.0", "first"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "admin_pocs.0", "POC-ARIN"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "tech_pocs.0", "POC-ARIN"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "routing_pocs.0", "POC-ARIN"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "remarks.0", "r1"),
				),
			},
			{
				Config: route("second", `[]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_route.test", "descriptions.0", "second"),
					resource.TestCheckResourceAttr("arin_irr_route.test", "remarks.#", "0"),
				),
			},
			{
				ResourceName:                         "arin_irr_route.test",
				ImportState:                          true,
				ImportStateId:                        "192.0.2.0/24/64496",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "prefix",
			},
		},
	})
}

func TestAccIRRRouteV6(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	cfg := providerConfig(srv.URL) + `
resource "arin_irr_route" "v6" {
  prefix       = "2001:db8::/32"
  origin_as    = 64496
  descriptions = ["v6"]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_route.v6", "ip_version", "6"),
					resource.TestCheckResourceAttr("arin_irr_route.v6", "prefix", "2001:db8::/32"),
				),
			},
		},
	})
}

func TestAccIRRRouteAutoLinkGuard(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	srv.InjectRoute(arin.IRRRoute{
		Prefix:              "198.51.100.0/24",
		OriginAS:            "AS64496",
		AutoLinkedRoaHandle: "roa-handle-1",
	})
	cfg := providerConfig(srv.URL) + `
resource "arin_irr_route" "guarded" {
  prefix       = "198.51.100.0/24"
  origin_as    = 64496
  descriptions = ["x"]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`auto.link|roa-handle-1`),
			},
		},
	})
}
