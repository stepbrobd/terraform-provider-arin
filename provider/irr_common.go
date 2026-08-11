package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// fromList converts a list value into a string slice, nil when null
func fromList(ctx context.Context, v types.List, diags *diag.Diagnostics) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	return out
}

// fromSet converts a set value into a string slice, nil when null
func fromSet(ctx context.Context, v types.Set, diags *diag.Diagnostics) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	return out
}

// toList renders a string slice as a list value, null for nil
func toList(ss []string) types.List {
	if ss == nil {
		return types.ListNull(types.StringType)
	}
	vals := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, vals)
}

// toSet renders a string slice as a set value, null for nil
func toSet(ss []string) types.Set {
	if ss == nil {
		return types.SetNull(types.StringType)
	}
	vals := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, vals)
}

// asString renders the wire form of an as number
func asString(n int64) string { return fmt.Sprintf("AS%d", n) }

// asNumber parses arin's AS-prefixed number rendering
func asNumber(s string, diags *diag.Diagnostics) int64 {
	n, err := strconv.ParseInt(strings.TrimPrefix(strings.ToUpper(s), "AS"), 10, 64)
	if err != nil {
		diags.AddError("unparseable as number", fmt.Sprintf("%q: %v", s, err))
	}
	return n
}

// canonPrefix normalizes arin's zero padded prefix rendering
func canonPrefix(s string) string {
	ip, length, ok := strings.Cut(s, "/")
	if !ok {
		return s
	}
	return canon(ip) + "/" + length
}

// samePrefix compares two prefixes semantically
func samePrefix(a, b string) bool {
	ipA, lenA, okA := strings.Cut(a, "/")
	ipB, lenB, okB := strings.Cut(b, "/")
	return okA && okB && lenA == lenB && sameAddr(ipA, ipB)
}

// pocRefs builds pocLinkRef entries for the AD, T, and R functions
func pocRefs(admin, tech, routing []string) []arin.PocLinkRef {
	var out []arin.PocLinkRef
	for _, h := range admin {
		out = append(out, arin.PocLinkRef{Function: "AD", Handle: h})
	}
	for _, h := range tech {
		out = append(out, arin.PocLinkRef{Function: "T", Handle: h})
	}
	for _, h := range routing {
		out = append(out, arin.PocLinkRef{Function: "R", Handle: h})
	}
	return out
}

// pocSplit separates pocLinkRef entries by function
func pocSplit(refs []arin.PocLinkRef) (admin, tech, routing []string) {
	for _, r := range refs {
		switch r.Function {
		case "AD":
			admin = append(admin, r.Handle)
		case "T":
			tech = append(tech, r.Handle)
		case "R":
			routing = append(routing, r.Handle)
		}
	}
	return
}

// nameRefs wraps names in arin's reference elements
func nameRefs(ss []string) []arin.NameRef {
	if ss == nil {
		return nil
	}
	out := make([]arin.NameRef, 0, len(ss))
	for _, s := range ss {
		out = append(out, arin.NameRef{Name: s})
	}
	return out
}

// names unwraps arin's reference elements
func names(refs []arin.NameRef) []string {
	if refs == nil {
		return nil
	}
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Name)
	}
	return out
}
