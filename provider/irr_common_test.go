package provider

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

func TestASConversions(t *testing.T) {
	var diags diag.Diagnostics
	if got := asNumber("AS10779", &diags); got != 10779 || diags.HasError() {
		t.Fatalf("asNumber = %d diags = %v", got, diags)
	}
	if got := asString(10779); got != "AS10779" {
		t.Fatalf("asString = %q", got)
	}
	asNumber("banana", &diags)
	if !diags.HasError() {
		t.Fatal("bad as accepted")
	}
}

func TestCanonPrefix(t *testing.T) {
	for in, want := range map[string]string{
		"023.161.104.000/24":                         "23.161.104.0/24",
		"2602:F590:0000:0000:0000:0000:0000:0000/36": "2602:f590::/36",
		"192.0.2.0/24":                               "192.0.2.0/24",
		"garbage":                                    "garbage",
	} {
		if got := canonPrefix(in); got != want {
			t.Fatalf("canonPrefix(%q) = %q want %q", in, got, want)
		}
	}
	if !samePrefix("023.161.104.000/24", "23.161.104.0/24") {
		t.Fatal("padded prefix not equal")
	}
	if samePrefix("192.0.2.0/24", "192.0.2.0/25") {
		t.Fatal("different lengths equal")
	}
}

func TestPocSplit(t *testing.T) {
	refs := []arin.PocLinkRef{{Function: "AD", Handle: "A1"}, {Function: "T", Handle: "T1"}, {Function: "T", Handle: "T2"}, {Function: "R", Handle: "R1"}}
	admin, tech, routing := pocSplit(refs)
	if !reflect.DeepEqual(admin, []string{"A1"}) || !reflect.DeepEqual(tech, []string{"T1", "T2"}) || !reflect.DeepEqual(routing, []string{"R1"}) {
		t.Fatalf("split = %v %v %v", admin, tech, routing)
	}
}

func TestNameRefs(t *testing.T) {
	refs := nameRefs([]string{"a", "b"})
	if !reflect.DeepEqual(names(refs), []string{"a", "b"}) {
		t.Fatalf("names = %v", names(refs))
	}
	if nameRefs(nil) != nil || names(nil) != nil {
		t.Fatal("nil not preserved")
	}
	var _ []arin.NameRef = refs
}
