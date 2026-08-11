package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestAccData(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	cfg := providerConfig(srv.URL) + `
resource "arin_roa" "a" {
  as_number = 64496
  name      = "headquarters"
  resources = [{
    start_address = "192.0.2.0"
    cidr_length   = 24
  }]
}

resource "arin_aspa" "a" {
  customer_as     = 64496
  provider_as_ids = [64497]
}

data "arin_roas" "all" {
  depends_on = [arin_roa.a]
}

data "arin_aspas" "all" {
  depends_on = [arin_aspa.a]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.arin_roas.all", "roas.#", "1"),
					resource.TestCheckResourceAttr("data.arin_roas.all", "roas.0.name", "headquarters"),
					resource.TestCheckResourceAttr("data.arin_roas.all", "roas.0.resources.0.end_address", "192.0.2.255"),
					resource.TestCheckResourceAttr("data.arin_aspas.all", "aspas.#", "1"),
					resource.TestCheckResourceAttr("data.arin_aspas.all", "aspas.0.customer_as", "64496"),
					resource.TestCheckResourceAttr("data.arin_aspas.all", "aspas.0.provider_as_ids.#", "1"),
				),
			},
		},
	})
}
