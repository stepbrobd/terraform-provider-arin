package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stepbrobd/terraform-provider-arin/arin"
	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func seedNet(srv *arintest.Server) {
	srv.InjectNet(arin.Net{
		Handle:           "NET-TEST-1",
		NetName:          "TESTNET",
		OrgHandle:        "TESTORG",
		ParentNetHandle:  "NET-PARENT",
		RegistrationDate: "2023-01-01T00:00:00-04:00",
		Version:          4,
		Comment:          arin.MakeLines([]string{"hello"}),
		PocLinks:         &arin.PocLinks{},
		OriginASes:       &arin.OriginASes{},
		NetBlocks: &arin.NetBlocks{Blocks: []arin.NetBlock{{
			CIDRLength:   24,
			Description:  "Direct Allocation",
			StartAddress: "192.000.002.000",
			EndAddress:   "192.000.002.255",
			Type:         "DA",
		}}},
	})
}

func TestAccNetAdopt(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	seedNet(srv)
	adopted := providerConfig(srv.URL) + `
resource "arin_net" "t" {
  handle = "NET-TEST-1"
}
`
	updated := providerConfig(srv.URL) + `
resource "arin_net" "t" {
  handle   = "NET-TEST-1"
  net_name = "TESTNET2"
  comment  = ["hello", "Geofeed https://example.com/geofeed.csv"]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config:             adopted,
				ResourceName:       "arin_net.t",
				ImportState:        true,
				ImportStateId:      "NET-TEST-1",
				ImportStatePersist: true,
			},
			{
				Config: adopted,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_net.t", "net_name", "TESTNET"),
					resource.TestCheckResourceAttr("arin_net.t", "comment.0", "hello"),
					resource.TestCheckResourceAttr("arin_net.t", "org_handle", "TESTORG"),
					resource.TestCheckResourceAttr("arin_net.t", "parent_net_handle", "NET-PARENT"),
					// the fake serves padded addresses, state is canonical
					resource.TestCheckResourceAttr("arin_net.t", "net_blocks.0.start_address", "192.0.2.0"),
					resource.TestCheckResourceAttr("arin_net.t", "net_blocks.0.type", "DA"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_net.t", "net_name", "TESTNET2"),
					resource.TestCheckResourceAttr("arin_net.t", "comment.1", "Geofeed https://example.com/geofeed.csv"),
				),
			},
		},
	})
}

func TestAccNetCreateRefused(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	seedNet(srv)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL) + `
resource "arin_net" "t" {
  handle = "NET-TEST-1"
}
`,
				ExpectError: regexp.MustCompile(`nets cannot be created`),
			},
		},
	})
}

func seedDelegation(srv *arintest.Server) {
	srv.InjectDelegation(arin.Delegation{
		Name:        "2.0.192.in-addr.arpa.",
		Nameservers: []string{"NS1.EXAMPLE.COM", "NS2.EXAMPLE.COM"},
		Keys: &arin.DelegationKeys{Keys: []arin.DelegationKey{{
			Algorithm:  arin.NamedValue{Name: "ECDSA Curve P-256 with SHA-256", Value: "13"},
			Digest:     "2935C3B989F1B43F47CCBE08DC45D0F827D359E15E8CD1EA1EC2D8398022A1D8",
			DigestType: arin.NamedValue{Name: "SHA-256", Value: "2"},
			KeyTag:     2371,
			TTL:        3600,
		}}},
	})
}

func TestAccDelegation(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	seedDelegation(srv)
	adopted := providerConfig(srv.URL) + `
resource "arin_delegation" "t" {
  name        = "2.0.192.in-addr.arpa"
  nameservers = ["NS1.EXAMPLE.COM", "NS2.EXAMPLE.COM"]
}
`
	updated := providerConfig(srv.URL) + `
resource "arin_delegation" "t" {
  name        = "2.0.192.in-addr.arpa"
  nameservers = ["NS1.EXAMPLE.COM", "NS2.EXAMPLE.COM", "NS3.EXAMPLE.COM"]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config:             adopted,
				ResourceName:       "arin_delegation.t",
				ImportState:        true,
				ImportStateId:      "2.0.192.in-addr.arpa",
				ImportStatePersist: true,
			},
			{
				Config: adopted,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arin_delegation.t", "name", "2.0.192.in-addr.arpa"),
					resource.TestCheckResourceAttr("arin_delegation.t", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("arin_delegation.t", "delegation_keys.#", "1"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr("arin_delegation.t", "nameservers.*", "NS3.EXAMPLE.COM"),
					// unconfigured keys ride along untouched
					resource.TestCheckResourceAttr("arin_delegation.t", "delegation_keys.#", "1"),
				),
			},
		},
	})
}

func TestAccDelegationCreateRefused(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	seedDelegation(srv)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL) + `
resource "arin_delegation" "t" {
  name        = "2.0.192.in-addr.arpa"
  nameservers = ["NS1.EXAMPLE.COM"]
}
`,
				ExpectError: regexp.MustCompile(`delegations cannot be created`),
			},
		},
	})
}

func TestAccRegistryData(t *testing.T) {
	srv := arintest.New(t, "TESTKEY", "TESTORG")
	seedNet(srv)
	seedDelegation(srv)
	srv.InjectOrg(arin.Org{
		Handle:  "TESTORG",
		OrgName: "Test Org, Inc.",
		City:    "Portland",
		PocLinks: &arin.PocLinks{Refs: []arin.PocLinkRef{
			{Function: "AD", Handle: "POC-ARIN"},
			{Function: "AB", Handle: "POC-ARIN"},
			{Function: "T", Handle: "POC-ARIN"},
		}},
		ISO3166One: arin.ISO3166One{Code2: "US"},
	})
	srv.InjectPoc(arin.Poc{
		Handle:      "POC-ARIN",
		ContactType: "ROLE",
		LastName:    "NOC",
		Emails:      []string{"noc@example.com"},
		Phones:      []arin.Phone{{Number: "+1-555-0100", Type: arin.PhoneType{Code: "O"}}},
		ISO3166One:  arin.ISO3166One{Code2: "US"},
	})
	cfg := providerConfig(srv.URL) + `
data "arin_net" "n" { handle = "NET-TEST-1" }
data "arin_delegation" "d" { name = "2.0.192.in-addr.arpa" }
data "arin_org" "o" {}
data "arin_poc" "p" { handle = "POC-ARIN" }
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.arin_net.n", "net_name", "TESTNET"),
					resource.TestCheckResourceAttr("data.arin_net.n", "net_blocks.0.end_address", "192.0.2.255"),
					resource.TestCheckResourceAttr("data.arin_delegation.d", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("data.arin_delegation.d", "delegation_keys.0.key_tag", "2371"),
					resource.TestCheckResourceAttr("data.arin_org.o", "org_name", "Test Org, Inc."),
					resource.TestCheckResourceAttr("data.arin_org.o", "country_code", "US"),
					resource.TestCheckResourceAttr("data.arin_org.o", "admin_pocs.0", "POC-ARIN"),
					resource.TestCheckResourceAttr("data.arin_poc.p", "last_name", "NOC"),
					resource.TestCheckResourceAttr("data.arin_poc.p", "phones.0.type", "O"),
				),
			},
		},
	})
}
