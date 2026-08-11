package arintest

import (
	"context"
	"testing"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

func TestIRRLifecycle(t *testing.T) {
	ctx := context.Background()
	srv := New(t, "KEY", "ORG")
	c, err := arin.New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}

	created, err := c.RouteCreate(ctx, "192.0.2.0/24", 64496, arin.IRRRoute{
		Prefix:      "192.0.2.0/24",
		OriginAS:    "AS64496",
		Description: arin.MakeLines([]string{"test route"}),
		PocLinks:    []arin.PocLinkRef{{Function: "AD", Handle: "EXA-ARIN"}, {Function: "T", Handle: "EXT-ARIN"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// the fake mirrors ote's padded rendering
	if created.Prefix != "192.000.002.000/24" || created.Version != 4 {
		t.Fatalf("created = %+v", created)
	}
	if created.NetHandle == "" || created.Created == "" || created.OrgHandle != "ORG" {
		t.Fatalf("created = %+v", created)
	}

	// duplicate create fails
	if _, err := c.RouteCreate(ctx, "192.0.2.0/24", 64496, arin.IRRRoute{Prefix: "192.0.2.0/24", OriginAS: "AS64496"}); err == nil {
		t.Fatal("duplicate create accepted")
	}

	got, err := c.Route(ctx, "192.0.2.0/24", 64496)
	if err != nil {
		t.Fatal(err)
	}
	firstMod := got.Modified

	upd, err := c.RouteUpdate(ctx, "192.0.2.0/24", 64496, arin.IRRRoute{
		Prefix:      "192.0.2.0/24",
		OriginAS:    "AS64496",
		Description: arin.MakeLines([]string{"renamed"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Modified == firstMod {
		t.Fatal("lastModified did not change on update")
	}
	if got := upd.Description.Strings()[0]; got != "renamed" {
		t.Fatalf("description = %q", got)
	}

	list, err := c.Routes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("routes = %+v err = %v", list, err)
	}

	if err := c.RouteDelete(ctx, "192.0.2.0/24", 64496); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Route(ctx, "192.0.2.0/24", 64496); !arin.IsNotFound(err) {
		t.Fatalf("after delete err = %v", err)
	}

	// aut-num and sets
	if _, err := c.AutNumCreate(ctx, 64496, arin.IRRAutNum{ASNumber: "AS64496", ASName: "EXAMPLE"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ASSetCreate(ctx, arin.IRRASSet{Name: "AS64496:AS-EXAMPLE", Members: []arin.NameRef{{Name: "AS64496"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RouteSetCreate(ctx, arin.IRRRouteSet{Name: "RS-EXAMPLE", MPMembers: []arin.NameRef{{Name: "2001:db8::/32"}}}); err != nil {
		t.Fatal(err)
	}
	sets, err := c.ASSets(ctx)
	if err != nil || len(sets) != 1 || sets[0].Name != "AS64496:AS-EXAMPLE" {
		t.Fatalf("assets = %+v err = %v", sets, err)
	}
	if _, err := c.RouteSet(ctx, "RS-EXAMPLE"); err != nil {
		t.Fatal(err)
	}
	if err := c.ASSetDelete(ctx, "AS64496:AS-EXAMPLE"); err != nil {
		t.Fatal(err)
	}
	if err := c.AutNumDelete(ctx, 64496); err != nil {
		t.Fatal(err)
	}
	if err := c.RouteSetDelete(ctx, "RS-EXAMPLE"); err != nil {
		t.Fatal(err)
	}
}

func TestIRRInjectAutoLinked(t *testing.T) {
	srv := New(t, "KEY", "ORG")
	c, err := arin.New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	srv.InjectRoute(arin.IRRRoute{
		Prefix:              "192.0.2.0/24",
		OriginAS:            "AS64496",
		AutoLinkedRoaHandle: "abc123",
	})
	got, err := c.Route(context.Background(), "192.0.2.0/24", 64496)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoLinkedRoaHandle != "abc123" {
		t.Fatalf("got = %+v", got)
	}
}
