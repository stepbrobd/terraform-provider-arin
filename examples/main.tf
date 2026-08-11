terraform {
  required_providers {
    arin = {
      source = "registry.terraform.io/stepbrobd/arin"
    }
  }
}

# api_key falls back to the ARIN_API_KEY environment variable
provider "arin" {
  org_handle = "EXAMPLE-ORG"
  base_url   = "https://reg.ote.arin.net" # drop for production
}

resource "arin_roa" "hq" {
  as_number = 64496
  name      = "headquarters"

  resources = [{
    start_address = "192.0.2.0"
    cidr_length   = 24
    max_length    = 24
  }]
}

resource "arin_aspa" "hq" {
  customer_as     = 64496
  provider_as_ids = [64497, 64498]
}

data "arin_roas" "all" {}

output "roa_handles" {
  value = data.arin_roas.all.roas[*].roa_handle
}
