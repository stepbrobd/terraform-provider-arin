package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccASPA(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	aspa := func(providers string) string {
		return providerConfig(srv.URL) + fmt.Sprintf(`
resource "arin_aspa" "test" {
  customer_as     = 64496
  provider_as_ids = [%s]
}
`, providers)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: aspa("64497, 64498"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_aspa.test", "customer_as", "64496"),
					resource.TestCheckResourceAttr("arin_aspa.test", "provider_as_ids.#", "2"),
				),
			},
			{
				Config: aspa("64499"),
				Check:  resource.TestCheckResourceAttr("arin_aspa.test", "provider_as_ids.#", "1"),
			},
			{
				ResourceName:                         "arin_aspa.test",
				ImportState:                          true,
				ImportStateId:                        "64496",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "customer_as",
			},
		},
	})
}
