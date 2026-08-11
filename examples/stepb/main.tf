# mirror of the STEPB org's rpki objects as read from ote on 2026-08-11
#
# applying this config as-is CREATES new objects alongside the existing
# ones (arin assigns fresh roa handles). to manage the existing objects
# instead, import them first:
#
#   tofu import arin_roa.as10779 <current roa handle for as10779>
#   tofu import arin_roa.as18932 <current roa handle for as18932>
#   tofu import arin_aspa.as10779 10779
#   tofu import arin_aspa.as18932 18932
#
# roa handles rotate on every arin-side renewal or update, so look the
# current ones up via the arin_roas data source before importing.

terraform {
  required_providers {
    arin = {
      source = "registry.terraform.io/stepbrobd/arin"
    }
  }
}

# api_key comes from the ARIN_API_KEY environment variable
provider "arin" {
  org_handle = "STEPB"
  base_url   = "https://reg.ote.arin.net" # drop for production
}

resource "arin_roa" "as10779" {
  as_number = 10779
  name      = ""
  auto_link = true

  resources = [
    {
      start_address = "2602:f590::"
      cidr_length   = 36
      max_length    = 48
    },
    {
      start_address = "23.161.104.0"
      cidr_length   = 24
      max_length    = 24
    },
    {
      start_address = "192.104.136.0"
      cidr_length   = 24
      max_length    = 24
    },
  ]
}

resource "arin_roa" "as18932" {
  as_number = 18932
  name      = ""
  auto_link = true

  resources = [
    {
      start_address = "2602:f590::"
      cidr_length   = 36
      max_length    = 48
    },
    {
      start_address = "23.161.104.0"
      cidr_length   = 24
      max_length    = 24
    },
    {
      start_address = "192.104.136.0"
      cidr_length   = 24
      max_length    = 24
    },
  ]
}

resource "arin_aspa" "as10779" {
  customer_as     = 10779
  provider_as_ids = [3204, 20473, 21700, 23961, 35661, 36236]
}

resource "arin_aspa" "as18932" {
  customer_as     = 18932
  provider_as_ids = [10779]
}
