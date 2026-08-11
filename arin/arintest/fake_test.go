package arintest

import (
	"context"
	"errors"
	"testing"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

func TestLifecycle(t *testing.T) {
	ctx := context.Background()
	srv := New(t, "KEY", "ORG")
	c, err := arin.New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}

	err = c.Transact(ctx, arin.Transaction{
		ROAAdds: &arin.ROAAdds{Specs: []arin.ROASpecRequest{{
			AutoLink: true,
			ASNumber: 64496,
			Name:     "headquarters",
			Resources: []arin.ROAResourceRequest{
				{StartAddress: "192.0.2.0", CIDRLength: 24, MaxLength: new(int64(25))},
				{StartAddress: "2001:db8::", CIDRLength: 32},
			},
		}}},
		ASPAAdds: &arin.ASPAAdds{ASPAs: []arin.ASPA{{CustomerASID: 64496, ProviderASIDs: []int64{64497}}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	roas, err := c.ROAs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(roas) != 1 {
		t.Fatalf("roas = %+v", roas)
	}
	roa := roas[0]
	if roa.Handle == "" || !roa.AutoRenewed || roa.NotValidBefore == "" {
		t.Fatalf("roa = %+v", roa)
	}
	if len(roa.Resources) != 2 {
		t.Fatalf("resources = %+v", roa.Resources)
	}
	if got := roa.Resources[0].EndAddress; got != "192.000.002.255" {
		t.Fatalf("v4 end = %q", got)
	}
	if got := roa.Resources[0].IPVersion; got != 4 {
		t.Fatalf("v4 version = %d", got)
	}
	if got := roa.Resources[1].EndAddress; got != "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff" {
		t.Fatalf("v6 end = %q", got)
	}
	if ml := roa.Resources[1].MaxLength; ml == nil || *ml != 32 {
		t.Fatalf("v6 default maxLength = %v", ml)
	}
	if !roa.Resources[0].AutoLinked {
		t.Fatal("autoLinked not propagated")
	}

	aspas, err := c.ASPAs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(aspas) != 1 || aspas[0].CustomerASID != 64496 {
		t.Fatalf("aspas = %+v", aspas)
	}

	err = c.Transact(ctx, arin.Transaction{
		ROADeletes:  &arin.ROADeletes{Handles: []arin.ROAHandleRef{{Handle: roa.Handle}}},
		ASPADeletes: &arin.ASPADeletes{CustomerASIDs: []int64{64496}},
	})
	if err != nil {
		t.Fatal(err)
	}
	roas, err = c.ROAs(ctx)
	if err != nil || len(roas) != 0 {
		t.Fatalf("after delete roas = %+v err = %v", roas, err)
	}

	// deleting a missing handle must fail atomically
	err = c.Transact(ctx, arin.Transaction{
		ROADeletes: &arin.ROADeletes{Handles: []arin.ROAHandleRef{{Handle: "nope"}}},
		ROAAdds:    &arin.ROAAdds{Specs: []arin.ROASpecRequest{{ASNumber: 1, Name: "x", Resources: []arin.ROAResourceRequest{{StartAddress: "192.0.2.0", CIDRLength: 24}}}}},
	})
	if !arin.IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
	roas, err = c.ROAs(ctx)
	if err != nil || len(roas) != 0 {
		t.Fatalf("failed transaction leaked state, roas = %+v err = %v", roas, err)
	}
}

func TestAuth(t *testing.T) {
	srv := New(t, "KEY", "ORG")
	c, err := arin.New(srv.URL, "WRONG", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ROAs(context.Background())
	var ae *arin.Error
	if !errors.As(err, &ae) || ae.Code != arin.CodeAuthentication {
		t.Fatalf("err = %v", err)
	}
}
