# Terraform Provider: ARIN

Manage ARIN RPKI objects (ROAs and ASPAs) with Terraform or OpenTofu via the
[ARIN RPKI RESTful API](https://www.arin.net/resources/manage/rpki/rpki-restful/).

Resources: `arin_roa`, `arin_aspa`. Data sources: `arin_roas`, `arin_aspas`. See
`examples/main.tf` for usage.

## Install

The provider is distributed as a nix package, not through the registry. Build it
and point `dev_overrides` at the result:

```sh
nix build
```

`~/.terraformrc` (or `~/.tofurc`):

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/stepbrobd/arin" = "/path/to/repo/result/bin"
  }
  direct {}
}
```

## Configuration

- `api_key`: ARIN API key, falls back to `ARIN_API_KEY`. Generate one in ARIN
  Online under Settings, Security Info, Manage API Keys.
- `org_handle`: org whose RPKI objects are managed. Use provider aliases for
  multiple orgs.
- `base_url`: defaults to `https://reg.arin.net`. Set `https://reg.ote.arin.net`
  for OT&E. OT&E data (API keys included) is a monthly snapshot of production,
  taken around the first Monday of each month.

## Development

```sh
nix develop
go test ./...

# acceptance tests against the in-memory fake
TF_ACC=1 go test ./provider/ -v
```

Updates to a ROA run as one atomic delete+add transaction on ARIN's side and
reissue the ROA under a new `roa_handle`.

Binary Cache:

- Cache: <https://cache.ysun.co>
- Key: `cache.ysun.co-1:WxPYwT5g3kt9XhUhHPpNLZKI9HIOsVVAuqSHpok8Qt4=`
