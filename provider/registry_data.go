package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// read-only registry lookups: net and delegation by identity, the
// provider org, and pocs by handle

// net

type netData struct {
	client *arin.Client
}

func newNetData() datasource.DataSource { return &netData{} }

func (d *netData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_net"
}

func (d *netData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *netData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	blocks := map[string]schema.Attribute{
		"start_address": schema.StringAttribute{Computed: true},
		"end_address":   schema.StringAttribute{Computed: true},
		"cidr_length":   schema.Int64Attribute{Computed: true},
		"description":   schema.StringAttribute{Computed: true},
		"type":          schema.StringAttribute{Computed: true},
	}
	resp.Schema = schema.Schema{
		Description: "One registry NET by handle.",
		Attributes: map[string]schema.Attribute{
			"handle":            schema.StringAttribute{Required: true},
			"net_name":          schema.StringAttribute{Computed: true},
			"comment":           schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"org_handle":        schema.StringAttribute{Computed: true},
			"parent_net_handle": schema.StringAttribute{Computed: true},
			"version":           schema.Int64Attribute{Computed: true},
			"registration_date": schema.StringAttribute{Computed: true},
			"origin_ases":       schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"net_blocks": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: blocks},
			},
		},
	}
}

func (d *netData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m netModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.Net(ctx, m.Handle.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("reading net", err.Error())
		return
	}
	m.refresh(got)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// delegation

type delegationData struct {
	client *arin.Client
}

func newDelegationData() datasource.DataSource { return &delegationData{} }

func (d *delegationData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_delegation"
}

func (d *delegationData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *delegationData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "One reverse DNS delegation by zone name.",
		Attributes: map[string]schema.Attribute{
			"name":        schema.StringAttribute{Required: true, Description: "Zone name, with or without the trailing dot."},
			"nameservers": schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"delegation_keys": schema.SetNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: delegationKeyDataAttrs()},
			},
		},
	}
}

func delegationKeyDataAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"algorithm":   schema.Int64Attribute{Computed: true},
		"digest":      schema.StringAttribute{Computed: true},
		"digest_type": schema.Int64Attribute{Computed: true},
		"key_tag":     schema.Int64Attribute{Computed: true},
		"ttl":         schema.Int64Attribute{Computed: true},
	}
}

func (d *delegationData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m delegationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.Delegation(ctx, m.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("reading delegation", err.Error())
		return
	}
	m.refresh(got, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// org

type orgData struct {
	client *arin.Client
}

func newOrgData() datasource.DataSource { return &orgData{} }

type orgModel struct {
	Handle              types.String `tfsdk:"handle"`
	OrgName             types.String `tfsdk:"org_name"`
	DBAName             types.String `tfsdk:"dba_name"`
	StreetAddress       types.List   `tfsdk:"street_address"`
	City                types.String `tfsdk:"city"`
	ISO3166Two          types.String `tfsdk:"iso3166_2"`
	PostalCode          types.String `tfsdk:"postal_code"`
	CountryCode         types.String `tfsdk:"country_code"`
	TaxID               types.String `tfsdk:"tax_id"`
	AcceptReassignments types.Bool   `tfsdk:"accept_reassignments"`
	Comment             types.List   `tfsdk:"comment"`
	RegistrationDate    types.String `tfsdk:"registration_date"`
	AdminPOCs           types.Set    `tfsdk:"admin_pocs"`
	AbusePOCs           types.Set    `tfsdk:"abuse_pocs"`
	DNSPOCs             types.Set    `tfsdk:"dns_pocs"`
	NOCPOCs             types.Set    `tfsdk:"noc_pocs"`
	RoutingPOCs         types.Set    `tfsdk:"routing_pocs"`
	TechPOCs            types.Set    `tfsdk:"tech_pocs"`
}

func (d *orgData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org"
}

func (d *orgData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *orgData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	strSet := schema.SetAttribute{Computed: true, ElementType: types.StringType}
	resp.Schema = schema.Schema{
		Description: "The provider org's registry record.",
		Attributes: map[string]schema.Attribute{
			"handle":               schema.StringAttribute{Computed: true},
			"org_name":             schema.StringAttribute{Computed: true},
			"dba_name":             schema.StringAttribute{Computed: true},
			"street_address":       schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"city":                 schema.StringAttribute{Computed: true},
			"iso3166_2":            schema.StringAttribute{Computed: true},
			"postal_code":          schema.StringAttribute{Computed: true},
			"country_code":         schema.StringAttribute{Computed: true},
			"tax_id":               schema.StringAttribute{Computed: true},
			"accept_reassignments": schema.BoolAttribute{Computed: true},
			"comment":              schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"registration_date":    schema.StringAttribute{Computed: true},
			"admin_pocs":           strSet,
			"abuse_pocs":           strSet,
			"dns_pocs":             strSet,
			"noc_pocs":             strSet,
			"routing_pocs":         strSet,
			"tech_pocs":            strSet,
		},
	}
}

// orgPocSplit separates org pocLinkRef entries by function
func orgPocSplit(refs []arin.PocLinkRef) map[string][]string {
	out := map[string][]string{}
	for _, r := range refs {
		out[r.Function] = append(out[r.Function], r.Handle)
	}
	return out
}

func (d *orgData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	got, err := d.client.Org(ctx, d.client.OrgHandle())
	if err != nil {
		resp.Diagnostics.AddError("reading org", err.Error())
		return
	}
	var refs []arin.PocLinkRef
	if got.PocLinks != nil {
		refs = got.PocLinks.Refs
	}
	fns := orgPocSplit(refs)
	m := orgModel{
		Handle:              types.StringValue(got.Handle),
		OrgName:             types.StringValue(got.OrgName),
		DBAName:             types.StringValue(got.DBAName),
		StreetAddress:       toList(got.StreetAddress.Strings()),
		City:                types.StringValue(got.City),
		ISO3166Two:          types.StringValue(got.ISO3166Two),
		PostalCode:          types.StringValue(got.PostalCode),
		CountryCode:         types.StringValue(got.ISO3166One.Code2),
		TaxID:               types.StringValue(got.TaxID),
		AcceptReassignments: types.BoolValue(got.AcceptReassignments),
		Comment:             toList(got.Comment.Strings()),
		RegistrationDate:    types.StringValue(got.RegistrationDate),
		AdminPOCs:           toSet(fns["AD"]),
		AbusePOCs:           toSet(fns["AB"]),
		DNSPOCs:             toSet(fns["D"]),
		NOCPOCs:             toSet(fns["N"]),
		RoutingPOCs:         toSet(fns["R"]),
		TechPOCs:            toSet(fns["T"]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// poc

type pocData struct {
	client *arin.Client
}

func newPocData() datasource.DataSource { return &pocData{} }

type phoneModel struct {
	Number types.String `tfsdk:"number"`
	Type   types.String `tfsdk:"type"`
}

type pocModel struct {
	Handle           types.String `tfsdk:"handle"`
	ContactType      types.String `tfsdk:"contact_type"`
	CompanyName      types.String `tfsdk:"company_name"`
	FirstName        types.String `tfsdk:"first_name"`
	LastName         types.String `tfsdk:"last_name"`
	StreetAddress    types.List   `tfsdk:"street_address"`
	City             types.String `tfsdk:"city"`
	ISO3166Two       types.String `tfsdk:"iso3166_2"`
	PostalCode       types.String `tfsdk:"postal_code"`
	CountryCode      types.String `tfsdk:"country_code"`
	Emails           types.Set    `tfsdk:"emails"`
	Phones           []phoneModel `tfsdk:"phones"`
	Comment          types.List   `tfsdk:"comment"`
	RegistrationDate types.String `tfsdk:"registration_date"`
}

func (d *pocData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_poc"
}

func (d *pocData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *pocData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "One registry point of contact by handle.",
		Attributes: map[string]schema.Attribute{
			"handle":            schema.StringAttribute{Required: true},
			"contact_type":      schema.StringAttribute{Computed: true},
			"company_name":      schema.StringAttribute{Computed: true},
			"first_name":        schema.StringAttribute{Computed: true},
			"last_name":         schema.StringAttribute{Computed: true},
			"street_address":    schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"city":              schema.StringAttribute{Computed: true},
			"iso3166_2":         schema.StringAttribute{Computed: true},
			"postal_code":       schema.StringAttribute{Computed: true},
			"country_code":      schema.StringAttribute{Computed: true},
			"emails":            schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"comment":           schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"registration_date": schema.StringAttribute{Computed: true},
			"phones": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"number": schema.StringAttribute{Computed: true},
					"type":   schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *pocData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m pocModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.Poc(ctx, m.Handle.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("reading poc", err.Error())
		return
	}
	m.Handle = types.StringValue(got.Handle)
	m.ContactType = types.StringValue(got.ContactType)
	m.CompanyName = types.StringValue(got.CompanyName)
	m.FirstName = types.StringValue(got.FirstName)
	m.LastName = types.StringValue(got.LastName)
	m.StreetAddress = toList(got.StreetAddress.Strings())
	m.City = types.StringValue(got.City)
	m.ISO3166Two = types.StringValue(got.ISO3166Two)
	m.PostalCode = types.StringValue(got.PostalCode)
	m.CountryCode = types.StringValue(got.ISO3166One.Code2)
	m.Emails = toSet(got.Emails)
	m.Phones = []phoneModel{}
	for _, p := range got.Phones {
		m.Phones = append(m.Phones, phoneModel{
			Number: types.StringValue(p.Number),
			Type:   types.StringValue(p.Type.Code),
		})
	}
	m.Comment = toList(got.Comment.Strings())
	m.RegistrationDate = types.StringValue(got.RegistrationDate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
