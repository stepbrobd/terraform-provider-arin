package provider

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type roaModel struct {
	ROAHandle      types.String       `tfsdk:"roa_handle"`
	ASNumber       types.Int64        `tfsdk:"as_number"`
	Name           types.String       `tfsdk:"name"`
	AutoLink       types.Bool         `tfsdk:"auto_link"`
	AutoRenewed    types.Bool         `tfsdk:"auto_renewed"`
	NotValidBefore types.String       `tfsdk:"not_valid_before"`
	NotValidAfter  types.String       `tfsdk:"not_valid_after"`
	Resources      []roaResourceModel `tfsdk:"resources"`
}

type roaResourceModel struct {
	StartAddress types.String `tfsdk:"start_address"`
	CIDRLength   types.Int64  `tfsdk:"cidr_length"`
	MaxLength    types.Int64  `tfsdk:"max_length"`
	EndAddress   types.String `tfsdk:"end_address"`
	IPVersion    types.Int64  `tfsdk:"ip_version"`
	AutoLinked   types.Bool   `tfsdk:"auto_linked"`
}

// spec converts the configured attributes into the arin request shape
func (m *roaModel) spec() arin.ROASpecRequest {
	out := arin.ROASpecRequest{
		AutoLink: m.AutoLink.ValueBool(),
		ASNumber: m.ASNumber.ValueInt64(),
		Name:     m.Name.ValueString(),
	}
	for _, r := range m.Resources {
		rr := arin.ROAResourceRequest{
			StartAddress: r.StartAddress.ValueString(),
			CIDRLength:   r.CIDRLength.ValueInt64(),
		}
		if !r.MaxLength.IsNull() {
			v := r.MaxLength.ValueInt64()
			rr.MaxLength = &v
		}
		out.Resources = append(out.Resources, rr)
	}
	return out
}

// parseAddr accepts arin's zero padded dotted quads in addition to
// canonical forms
func parseAddr(s string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(s); err == nil {
		return a, true
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return netip.Addr{}, false
	}
	var b [4]byte
	for i, p := range parts {
		if p == "" || len(p) > 3 {
			return netip.Addr{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return netip.Addr{}, false
		}
		b[i] = byte(n)
	}
	return netip.AddrFrom4(b), true
}

// sameAddr compares two textual ip addresses semantically
// arin may return a different textual form than the configuration
func sameAddr(a, b string) bool {
	x, okX := parseAddr(a)
	y, okY := parseAddr(b)
	return okX && okY && x == y
}

// canon returns the canonical text form of an address when parseable
func canon(s string) string {
	if a, ok := parseAddr(s); ok {
		return a.String()
	}
	return s
}

// matches reports whether s carries the configured identity of m
// identity is as number, name, and the set of start/cidr pairs
func matches(s *arin.ROASpec, m *roaModel) bool {
	if s.ASNumber != m.ASNumber.ValueInt64() || s.Name != m.Name.ValueString() {
		return false
	}
	if len(s.Resources) != len(m.Resources) {
		return false
	}
	used := make([]bool, len(s.Resources))
outer:
	for _, r := range m.Resources {
		for i := range s.Resources {
			if used[i] {
				continue
			}
			if sameAddr(s.Resources[i].StartAddress, r.StartAddress.ValueString()) &&
				s.Resources[i].CIDRLength == r.CIDRLength.ValueInt64() {
				used[i] = true
				continue outer
			}
		}
		return false
	}
	return true
}

// errSettleNoMatch reports a listing that does not yet show the new
// roa, which can be the listing lagging the transaction
var errSettleNoMatch = errors.New("transaction succeeded but no new roa matches the configuration")

// findNew locates the roa created by the previous transaction
// arin assigns handles server side and the post response is
// undocumented, so creation is identified by diffing listings taken
// around the transaction
func findNew(before map[string]bool, after []arin.ROASpec, m *roaModel) (*arin.ROASpec, error) {
	var hits []*arin.ROASpec
	for i := range after {
		if before[after[i].Handle] || !matches(&after[i], m) {
			continue
		}
		hits = append(hits, &after[i])
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return nil, errSettleNoMatch
	default:
		return nil, fmt.Errorf("%d new roas match the configuration, cannot identify ours", len(hits))
	}
}

// refresh overwrites m with the server state s, preserving configured
// textual forms and list order when semantically unchanged
func (m *roaModel) refresh(s *arin.ROASpec) {
	m.ROAHandle = types.StringValue(s.Handle)
	m.ASNumber = types.Int64Value(s.ASNumber)
	m.Name = types.StringValue(s.Name)
	m.AutoRenewed = types.BoolValue(s.AutoRenewed)
	m.NotValidBefore = types.StringValue(s.NotValidBefore)
	m.NotValidAfter = types.StringValue(s.NotValidAfter)

	next := make([]roaResourceModel, 0, len(s.Resources))
	used := make([]bool, len(s.Resources))
	aligned := len(s.Resources) == len(m.Resources)
	if aligned {
	outer:
		for _, r := range m.Resources {
			for i := range s.Resources {
				if used[i] {
					continue
				}
				if sameAddr(s.Resources[i].StartAddress, r.StartAddress.ValueString()) &&
					s.Resources[i].CIDRLength == r.CIDRLength.ValueInt64() {
					used[i] = true
					next = append(next, mergeResource(r, &s.Resources[i]))
					continue outer
				}
			}
			aligned = false
			break
		}
	}
	if !aligned {
		next = next[:0]
		for i := range s.Resources {
			next = append(next, serverResource(&s.Resources[i]))
		}
	}
	m.Resources = next
}

// mergeResource keeps the configured textual start address and the
// configured absence of max length when the server defaulted it
func mergeResource(state roaResourceModel, s *arin.ROAResource) roaResourceModel {
	out := roaResourceModel{
		StartAddress: state.StartAddress,
		CIDRLength:   types.Int64Value(s.CIDRLength),
		EndAddress:   types.StringValue(canon(s.EndAddress)),
		IPVersion:    types.Int64Value(s.IPVersion),
		AutoLinked:   types.BoolValue(s.AutoLinked),
	}
	switch {
	case s.MaxLength == nil:
		out.MaxLength = types.Int64Null()
	case state.MaxLength.IsNull() && *s.MaxLength == s.CIDRLength:
		// the server may default max length to cidr length
		// keep the configured null to avoid a permanent diff
		out.MaxLength = types.Int64Null()
	default:
		out.MaxLength = types.Int64Value(*s.MaxLength)
	}
	return out
}

func serverResource(s *arin.ROAResource) roaResourceModel {
	out := roaResourceModel{
		StartAddress: types.StringValue(canon(s.StartAddress)),
		CIDRLength:   types.Int64Value(s.CIDRLength),
		EndAddress:   types.StringValue(canon(s.EndAddress)),
		IPVersion:    types.Int64Value(s.IPVersion),
		AutoLinked:   types.BoolValue(s.AutoLinked),
		MaxLength:    types.Int64Null(),
	}
	if s.MaxLength != nil {
		out.MaxLength = types.Int64Value(*s.MaxLength)
	}
	return out
}
