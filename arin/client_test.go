package arin

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestTransactRequest(t *testing.T) {
	var body []byte
	var hdr http.Header
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		hdr = r.Header.Clone()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		body = b
	}))
	defer srv.Close()

	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	tx := Transaction{
		ROADeletes: &ROADeletes{Handles: []ROAHandleRef{{AutoLink: true, Handle: "24ab90ed"}}},
		ROAAdds: &ROAAdds{Specs: []ROASpecRequest{{
			AutoLink: true,
			ASNumber: 64496,
			Name:     "headquarters",
			Resources: []ROAResourceRequest{{
				StartAddress: "192.0.2.0",
				CIDRLength:   24,
				MaxLength:    ptr(int64(25)),
			}},
		}}},
		ASPAAdds: &ASPAAdds{ASPAs: []ASPA{{CustomerASID: 64496, ProviderASIDs: []int64{64497, 64498}}}},
	}
	if err := c.Transact(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if path != "/rest/rpki/ORG" {
		t.Fatalf("path = %q", path)
	}
	if h := hdr.Get("Authorization"); h != "ApiKey KEY" {
		t.Fatalf("authorization = %q", h)
	}
	if ct := hdr.Get("Content-Type"); ct != "application/xml" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(string(body), `<rpkiTransaction xmlns="http://www.arin.net/regrws/rpki/v1">`) {
		t.Fatalf("missing namespaced root in %s", body)
	}
	if !strings.Contains(string(body), `<roaHandle autolink="true">24ab90ed</roaHandle>`) {
		t.Fatalf("missing delete handle in %s", body)
	}
	var back Transaction
	if err := xml.Unmarshal(body, &back); err != nil {
		t.Fatal(err)
	}
	back.XMLName = xml.Name{}
	for i := range back.ASPAAdds.ASPAs {
		back.ASPAAdds.ASPAs[i].XMLName = xml.Name{}
	}
	if !reflect.DeepEqual(tx, back) {
		t.Fatalf("round trip mismatch\nsent: %+v\ngot:  %+v", tx, back)
	}
}

func TestNewRejectsBadInput(t *testing.T) {
	for _, tc := range []struct{ base, key, org string }{
		{"https://reg.arin.net", "", "ORG"},
		{"https://reg.arin.net", "KEY", ""},
		{"://bad", "KEY", "ORG"},
		{"reg.arin.net", "KEY", "ORG"},
	} {
		if _, err := New(tc.base, tc.key, tc.org); err == nil {
			t.Errorf("New(%q, %q, %q) accepted", tc.base, tc.key, tc.org)
		}
	}
}
