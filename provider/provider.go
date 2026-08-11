// Package provider implements the terraform provider for arin rpki
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type arinProvider struct {
	version string
}

// New returns a provider factory carrying the build version
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &arinProvider{version: version}
	}
}

type providerModel struct {
	APIKey    types.String `tfsdk:"api_key"`
	OrgHandle types.String `tfsdk:"org_handle"`
	BaseURL   types.String `tfsdk:"base_url"`
}

func (p *arinProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "arin"
	resp.Version = p.version
}

func (p *arinProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage ARIN RPKI objects (ROAs and ASPAs) through the RPKI RESTful API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "ARIN API key. Falls back to the ARIN_API_KEY environment variable.",
			},
			"org_handle": schema.StringAttribute{
				Required:    true,
				Description: "Org handle owning the RPKI objects. Use provider aliases for multiple orgs.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "API base URL. Defaults to " + arin.DefaultBaseURL + ". Set to " + arin.OTEBaseURL + " for OT&E.",
			},
		},
	}
}

func (p *arinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cfg.APIKey.IsUnknown() || cfg.OrgHandle.IsUnknown() || cfg.BaseURL.IsUnknown() {
		resp.Diagnostics.AddError(
			"unknown provider configuration",
			"api_key, org_handle, and base_url must be known at plan time",
		)
		return
	}
	key := os.Getenv("ARIN_API_KEY")
	if !cfg.APIKey.IsNull() {
		key = cfg.APIKey.ValueString()
	}
	if key == "" {
		resp.Diagnostics.AddError(
			"missing api key",
			"set the api_key attribute or the ARIN_API_KEY environment variable",
		)
		return
	}
	base := arin.DefaultBaseURL
	if !cfg.BaseURL.IsNull() {
		base = cfg.BaseURL.ValueString()
	}
	client, err := arin.New(base, key, cfg.OrgHandle.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid provider configuration", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *arinProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newROAResource, newASPAResource}
}

func (p *arinProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newROAsData, newASPAsData}
}

// clientFrom extracts the configured client from provider data
// a nil return with no diagnostics means configure has not run yet
func clientFrom(data any, diags *diag.Diagnostics) *arin.Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*arin.Client)
	if !ok {
		diags.AddError("unexpected provider data", fmt.Sprintf("expected *arin.Client, got %T", data))
		return nil
	}
	return c
}
