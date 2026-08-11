package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// shared nested entry for pocs and text carried by every irr class

type irrRouteEntry struct {
	Prefix              types.String `tfsdk:"prefix"`
	OriginAS            types.Int64  `tfsdk:"origin_as"`
	Descriptions        types.List   `tfsdk:"descriptions"`
	AdminPOCs           types.Set    `tfsdk:"admin_pocs"`
	TechPOCs            types.Set    `tfsdk:"tech_pocs"`
	RoutingPOCs         types.Set    `tfsdk:"routing_pocs"`
	Remarks             types.List   `tfsdk:"remarks"`
	MemberOf            types.Set    `tfsdk:"member_of"`
	IPVersion           types.Int64  `tfsdk:"ip_version"`
	NetHandle           types.String `tfsdk:"net_handle"`
	AutoLinkedROAHandle types.String `tfsdk:"auto_linked_roa_handle"`
	Created             types.String `tfsdk:"created"`
	LastModified        types.String `tfsdk:"last_modified"`
}

func irrTextAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"descriptions": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"admin_pocs":   schema.SetAttribute{Computed: true, ElementType: types.StringType},
		"tech_pocs":    schema.SetAttribute{Computed: true, ElementType: types.StringType},
		"routing_pocs": schema.SetAttribute{Computed: true, ElementType: types.StringType},
		"remarks":      schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"created":      schema.StringAttribute{Computed: true},
		"last_modified": schema.StringAttribute{
			Computed: true,
		},
	}
}

// routes

type irrRoutesData struct {
	client *arin.Client
}

func newIRRRoutesData() datasource.DataSource { return &irrRoutesData{} }

type irrRoutesModel struct {
	Routes []irrRouteEntry `tfsdk:"routes"`
}

func (d *irrRoutesData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_routes"
}

func (d *irrRoutesData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *irrRoutesData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := irrTextAttrs()
	attrs["prefix"] = schema.StringAttribute{Computed: true}
	attrs["origin_as"] = schema.Int64Attribute{Computed: true}
	attrs["member_of"] = schema.SetAttribute{Computed: true, ElementType: types.StringType}
	attrs["ip_version"] = schema.Int64Attribute{Computed: true}
	attrs["net_handle"] = schema.StringAttribute{Computed: true}
	attrs["auto_linked_roa_handle"] = schema.StringAttribute{Computed: true}
	resp.Schema = schema.Schema{
		Description: "All IRR route objects under the provider org, auto-linked ones included.",
		Attributes: map[string]schema.Attribute{
			"routes": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: attrs},
			},
		},
	}
}

func (d *irrRoutesData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.Routes(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing irr routes", err.Error())
		return
	}
	out := irrRoutesModel{Routes: make([]irrRouteEntry, 0, len(list))}
	for i := range list {
		r := &list[i]
		admin, tech, routing := pocSplit(r.PocLinks)
		var diags diag.Diagnostics
		e := irrRouteEntry{
			Prefix:              types.StringValue(canonPrefix(r.Prefix)),
			OriginAS:            types.Int64Value(asNumber(r.OriginAS, &diags)),
			Descriptions:        toList(r.Description.Strings()),
			AdminPOCs:           toSet(admin),
			TechPOCs:            toSet(tech),
			RoutingPOCs:         toSet(routing),
			Remarks:             toList(r.Remarks.Strings()),
			MemberOf:            toSet(names(r.MemberOf)),
			IPVersion:           types.Int64Value(r.Version),
			NetHandle:           types.StringValue(r.NetHandle),
			AutoLinkedROAHandle: types.StringValue(r.AutoLinkedRoaHandle),
			Created:             types.StringValue(r.Created),
			LastModified:        types.StringValue(r.Modified),
		}
		resp.Diagnostics.Append(diags...)
		out.Routes = append(out.Routes, e)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

// aut-nums

type irrAutNumsData struct {
	client *arin.Client
}

func newIRRAutNumsData() datasource.DataSource { return &irrAutNumsData{} }

type irrAutNumEntry struct {
	ASNumber     types.Int64  `tfsdk:"as_number"`
	ASName       types.String `tfsdk:"as_name"`
	Descriptions types.List   `tfsdk:"descriptions"`
	AdminPOCs    types.Set    `tfsdk:"admin_pocs"`
	TechPOCs     types.Set    `tfsdk:"tech_pocs"`
	RoutingPOCs  types.Set    `tfsdk:"routing_pocs"`
	Remarks      types.List   `tfsdk:"remarks"`
	Imports      types.List   `tfsdk:"imports"`
	Exports      types.List   `tfsdk:"exports"`
	Defaults     types.List   `tfsdk:"defaults"`
	MPImports    types.List   `tfsdk:"mp_imports"`
	MPExports    types.List   `tfsdk:"mp_exports"`
	MPDefaults   types.List   `tfsdk:"mp_defaults"`
	MemberOf     types.Set    `tfsdk:"member_of"`
	Created      types.String `tfsdk:"created"`
	LastModified types.String `tfsdk:"last_modified"`
}

type irrAutNumsModel struct {
	AutNums []irrAutNumEntry `tfsdk:"aut_nums"`
}

func (d *irrAutNumsData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_aut_nums"
}

func (d *irrAutNumsData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *irrAutNumsData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := irrTextAttrs()
	attrs["as_number"] = schema.Int64Attribute{Computed: true}
	attrs["as_name"] = schema.StringAttribute{Computed: true}
	for _, k := range []string{"imports", "exports", "defaults", "mp_imports", "mp_exports", "mp_defaults"} {
		attrs[k] = schema.ListAttribute{Computed: true, ElementType: types.StringType}
	}
	attrs["member_of"] = schema.SetAttribute{Computed: true, ElementType: types.StringType}
	resp.Schema = schema.Schema{
		Description: "All IRR aut-num objects under the provider org.",
		Attributes: map[string]schema.Attribute{
			"aut_nums": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: attrs},
			},
		},
	}
}

func (d *irrAutNumsData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.AutNums(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing irr aut-nums", err.Error())
		return
	}
	out := irrAutNumsModel{AutNums: make([]irrAutNumEntry, 0, len(list))}
	for i := range list {
		a := &list[i]
		admin, tech, routing := pocSplit(a.PocLinks)
		var diags diag.Diagnostics
		out.AutNums = append(out.AutNums, irrAutNumEntry{
			ASNumber:     types.Int64Value(asNumber(a.ASNumber, &diags)),
			ASName:       types.StringValue(a.ASName),
			Descriptions: toList(a.Description.Strings()),
			AdminPOCs:    toSet(admin),
			TechPOCs:     toSet(tech),
			RoutingPOCs:  toSet(routing),
			Remarks:      toList(a.Remarks.Strings()),
			Imports:      toList(a.Imports.Strings()),
			Exports:      toList(a.Exports.Strings()),
			Defaults:     toList(a.Defaults.Strings()),
			MPImports:    toList(a.MPImports.Strings()),
			MPExports:    toList(a.MPExports.Strings()),
			MPDefaults:   toList(a.MPDefaults.Strings()),
			MemberOf:     toSet(names(a.MemberOf)),
			Created:      types.StringValue(a.Created),
			LastModified: types.StringValue(a.Modified),
		})
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

// as-sets and route-sets

type irrSetEntry struct {
	Name         types.String `tfsdk:"name"`
	Descriptions types.List   `tfsdk:"descriptions"`
	AdminPOCs    types.Set    `tfsdk:"admin_pocs"`
	TechPOCs     types.Set    `tfsdk:"tech_pocs"`
	RoutingPOCs  types.Set    `tfsdk:"routing_pocs"`
	Remarks      types.List   `tfsdk:"remarks"`
	Members      types.Set    `tfsdk:"members"`
	MbrsByRef    types.Set    `tfsdk:"mbrs_by_ref"`
	Created      types.String `tfsdk:"created"`
	LastModified types.String `tfsdk:"last_modified"`
}

type irrRouteSetEntry struct {
	irrSetEntry
	MPMembers types.Set `tfsdk:"mp_members"`
}

type irrASSetsData struct {
	client *arin.Client
}

func newIRRASSetsData() datasource.DataSource { return &irrASSetsData{} }

type irrASSetsModel struct {
	ASSets []irrSetEntry `tfsdk:"as_sets"`
}

func irrSetEntryAttrs() map[string]schema.Attribute {
	attrs := irrTextAttrs()
	attrs["name"] = schema.StringAttribute{Computed: true}
	attrs["members"] = schema.SetAttribute{Computed: true, ElementType: types.StringType}
	attrs["mbrs_by_ref"] = schema.SetAttribute{Computed: true, ElementType: types.StringType}
	return attrs
}

func (d *irrASSetsData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_as_sets"
}

func (d *irrASSetsData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *irrASSetsData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All IRR as-set objects under the provider org.",
		Attributes: map[string]schema.Attribute{
			"as_sets": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: irrSetEntryAttrs()},
			},
		},
	}
}

func setEntry(name string, descr, remarks *arin.Lines, links []arin.PocLinkRef, members, byRef []arin.NameRef, created, modified string) irrSetEntry {
	admin, tech, routing := pocSplit(links)
	return irrSetEntry{
		Name:         types.StringValue(name),
		Descriptions: toList(descr.Strings()),
		AdminPOCs:    toSet(admin),
		TechPOCs:     toSet(tech),
		RoutingPOCs:  toSet(routing),
		Remarks:      toList(remarks.Strings()),
		Members:      toSet(names(members)),
		MbrsByRef:    toSet(names(byRef)),
		Created:      types.StringValue(created),
		LastModified: types.StringValue(modified),
	}
}

func (d *irrASSetsData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.ASSets(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing irr as-sets", err.Error())
		return
	}
	out := irrASSetsModel{ASSets: make([]irrSetEntry, 0, len(list))}
	for i := range list {
		s := &list[i]
		out.ASSets = append(out.ASSets, setEntry(s.Name, s.Description, s.Remarks, s.PocLinks, s.Members, s.MembersByRef, s.Created, s.Modified))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

type irrRouteSetsData struct {
	client *arin.Client
}

func newIRRRouteSetsData() datasource.DataSource { return &irrRouteSetsData{} }

type irrRouteSetsModel struct {
	RouteSets []irrRouteSetEntry `tfsdk:"route_sets"`
}

func (d *irrRouteSetsData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_route_sets"
}

func (d *irrRouteSetsData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *irrRouteSetsData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := irrSetEntryAttrs()
	attrs["mp_members"] = schema.SetAttribute{Computed: true, ElementType: types.StringType}
	resp.Schema = schema.Schema{
		Description: "All IRR route-set objects under the provider org.",
		Attributes: map[string]schema.Attribute{
			"route_sets": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: attrs},
			},
		},
	}
}

func (d *irrRouteSetsData) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.RouteSets(ctx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("listing irr route-sets", err.Error())
		return
	}
	out := irrRouteSetsModel{RouteSets: make([]irrRouteSetEntry, 0, len(list))}
	for i := range list {
		s := &list[i]
		out.RouteSets = append(out.RouteSets, irrRouteSetEntry{
			irrSetEntry: setEntry(s.Name, s.Description, s.Remarks, s.PocLinks, s.Members, s.MembersByRef, s.Created, s.Modified),
			MPMembers:   toSet(names(s.MPMembers)),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
