package provider_test

import (
	"context"
	"fmt"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/stepbrobd/terraform-provider-arin/provider"
)

// factories serves the provider in-process for acceptance tests
func factories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"arin": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// providerConfig points the provider at a fake server
func providerConfig(url string) string {
	return fmt.Sprintf(`
provider "arin" {
  api_key    = "TESTKEY"
  org_handle = "TESTORG"
  base_url   = %q
}
`, url)
}

func TestSchemas(t *testing.T) {
	ctx := context.Background()
	p := provider.New("test")()

	var resp fwprovider.SchemaResponse
	p.Schema(ctx, fwprovider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	var meta fwprovider.MetadataResponse
	p.Metadata(ctx, fwprovider.MetadataRequest{}, &meta)
	if meta.TypeName != "arin" || meta.Version != "test" {
		t.Fatalf("metadata = %+v", meta)
	}

	for _, f := range p.Resources(ctx) {
		r := f()
		var s fwresource.SchemaResponse
		r.Schema(ctx, fwresource.SchemaRequest{}, &s)
		if s.Diagnostics.HasError() {
			t.Fatal(s.Diagnostics)
		}
	}
	for _, f := range p.DataSources(ctx) {
		d := f()
		var s fwdatasource.SchemaResponse
		d.Schema(ctx, fwdatasource.SchemaRequest{}, &s)
		if s.Diagnostics.HasError() {
			t.Fatal(s.Diagnostics)
		}
	}
}
