package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type aspasData struct {
	client *arin.Client
}

func newASPAsData() datasource.DataSource { return &aspasData{} }

type aspasDataModel struct {
	ASPAs []aspaEntry `tfsdk:"aspas"`
}

type aspaEntry struct {
	CustomerAS    types.Int64 `tfsdk:"customer_as"`
	ProviderASIDs []int64     `tfsdk:"provider_as_ids"`
}

func (d *aspasData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aspas"
}

func (d *aspasData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *aspasData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All ASPAs registered under the provider org.",
		Attributes: map[string]schema.Attribute{
			"aspas": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"customer_as": schema.Int64Attribute{Computed: true},
						"provider_as_ids": schema.SetAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
	}
}

func (d *aspasData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.ASPAs(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing aspas", err.Error())
		return
	}
	out := aspasDataModel{ASPAs: make([]aspaEntry, 0, len(list))}
	for i := range list {
		out.ASPAs = append(out.ASPAs, aspaEntry{
			CustomerAS:    types.Int64Value(list[i].CustomerASID),
			ProviderASIDs: list[i].ProviderASIDs,
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
