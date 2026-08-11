package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/stepbrobd/terraform-provider-arin/arin"
	"github.com/stepbrobd/terraform-provider-arin/arin/arintest"
)

func TestRouteMutable(t *testing.T) {
	srv := arintest.New(t, "KEY", "ORG")
	c, err := arin.New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	r := &irrRouteResource{client: c}
	ctx := context.Background()

	exists, err := r.mutable(ctx, "192.0.2.0/24", 64496)
	if err != nil || exists {
		t.Fatalf("absent route: exists=%v err=%v", exists, err)
	}

	srv.InjectRoute(arin.IRRRoute{Prefix: "192.0.2.0/24", OriginAS: "AS64496"})
	exists, err = r.mutable(ctx, "192.0.2.0/24", 64496)
	if err != nil || !exists {
		t.Fatalf("plain route: exists=%v err=%v", exists, err)
	}

	srv.InjectRoute(arin.IRRRoute{Prefix: "198.51.100.0/24", OriginAS: "AS64496", AutoLinkedRoaHandle: "roa-1"})
	_, err = r.mutable(ctx, "198.51.100.0/24", 64496)
	if err == nil || !strings.Contains(err.Error(), "auto-linked") {
		t.Fatalf("auto-linked route: err=%v", err)
	}
	// the fake stores the padded rendering, the guard message must
	// carry the canonical form
	if !strings.Contains(err.Error(), "198.51.100.0/24") {
		t.Fatalf("guard message not canonical: %v", err)
	}
}
