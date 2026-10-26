package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin"
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
				// member-of is set outside terraform and the update must keep it
				PreConfig: func() {
					ctx := context.Background()
					c, err := arin.New(srv.URL, "TESTKEY", "TESTORG")
					if err != nil {
						t.Fatal(err)
					}
					a, err := c.AutNum(ctx, 64496)
					if err != nil {
						t.Fatal(err)
					}
					a.PocLinks = nil
					a.MemberOf = []arin.NameRef{{Name: "AS64496:AS-TEST"}}
					if _, err := c.AutNumUpdate(ctx, 64496, *a); err != nil {
						t.Fatal(err)
					}
				},
				Config: autnum("RENAMED", "\n  mp_imports   = [\"afi any from AS64497 accept ANY\"]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "as_name", "RENAMED"),
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "mp_imports.0", "afi any from AS64497 accept ANY"),
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "member_of.#", "1"),
					resource.TestCheckResourceAttr("arin_irr_aut_num.test", "member_of.0", "AS64496:AS-TEST"),
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
