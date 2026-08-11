package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type roasData struct {
	client *arin.Client
}

func newROAsData() datasource.DataSource { return &roasData{} }

type roasDataModel struct {
	ROAs []roaEntry `tfsdk:"roas"`
}

type roaEntry struct {
	ROAHandle      types.String       `tfsdk:"roa_handle"`
	ASNumber       types.Int64        `tfsdk:"as_number"`
	Name           types.String       `tfsdk:"name"`
	AutoRenewed    types.Bool         `tfsdk:"auto_renewed"`
	NotValidBefore types.String       `tfsdk:"not_valid_before"`
	NotValidAfter  types.String       `tfsdk:"not_valid_after"`
	Resources      []roaEntryResource `tfsdk:"resources"`
}

type roaEntryResource struct {
	StartAddress types.String `tfsdk:"start_address"`
	EndAddress   types.String `tfsdk:"end_address"`
	CIDRLength   types.Int64  `tfsdk:"cidr_length"`
	MaxLength    types.Int64  `tfsdk:"max_length"`
	IPVersion    types.Int64  `tfsdk:"ip_version"`
	AutoLinked   types.Bool   `tfsdk:"auto_linked"`
}

func (d *roasData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roas"
}

func (d *roasData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *roasData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All ROAs registered under the provider org.",
		Attributes: map[string]schema.Attribute{
			"roas": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"roa_handle":       schema.StringAttribute{Computed: true},
						"as_number":        schema.Int64Attribute{Computed: true},
						"name":             schema.StringAttribute{Computed: true},
						"auto_renewed":     schema.BoolAttribute{Computed: true},
						"not_valid_before": schema.StringAttribute{Computed: true},
						"not_valid_after":  schema.StringAttribute{Computed: true},
						"resources": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"start_address": schema.StringAttribute{Computed: true},
									"end_address":   schema.StringAttribute{Computed: true},
									"cidr_length":   schema.Int64Attribute{Computed: true},
									"max_length":    schema.Int64Attribute{Computed: true},
									"ip_version":    schema.Int64Attribute{Computed: true},
									"auto_linked":   schema.BoolAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *roasData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.ROAs(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing roas", err.Error())
		return
	}
	out := roasDataModel{ROAs: make([]roaEntry, 0, len(list))}
	for i := range list {
		s := &list[i]
		e := roaEntry{
			ROAHandle:      types.StringValue(s.Handle),
			ASNumber:       types.Int64Value(s.ASNumber),
			Name:           types.StringValue(s.Name),
			AutoRenewed:    types.BoolValue(s.AutoRenewed),
			NotValidBefore: types.StringValue(s.NotValidBefore),
			NotValidAfter:  types.StringValue(s.NotValidAfter),
			Resources:      make([]roaEntryResource, 0, len(s.Resources)),
		}
		for j := range s.Resources {
			r := &s.Resources[j]
			er := roaEntryResource{
				StartAddress: types.StringValue(r.StartAddress),
				EndAddress:   types.StringValue(r.EndAddress),
				CIDRLength:   types.Int64Value(r.CIDRLength),
				IPVersion:    types.Int64Value(r.IPVersion),
				AutoLinked:   types.BoolValue(r.AutoLinked),
				MaxLength:    types.Int64Null(),
			}
			if r.MaxLength != nil {
				er.MaxLength = types.Int64Value(*r.MaxLength)
			}
			e.Resources = append(e.Resources, er)
		}
		out.ROAs = append(out.ROAs, e)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
