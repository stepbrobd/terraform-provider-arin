package arin

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestTransactRequest(t *testing.T) {
	var body []byte
	var hdr http.Header
	var path string
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query().Get("apikey")
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
				MaxLength:    new(int64(25)),
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
	if query != "KEY" {
		t.Fatalf("apikey query param = %q", query)
	}
	if ct := hdr.Get("Content-Type"); ct != "application/xml" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(string(body), `<rpkiTransaction xmlns="http://www.arin.net/regrws/rpki/v1">`) {
		t.Fatalf("missing namespaced root in %s", body)
	}
	if !strings.Contains(string(body), `<roaHandle autoLink="true">24ab90ed</roaHandle>`) {
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

// wrapped in a hypothetical collection envelope
// the real envelope is undocumented and decode must not depend on it
const roaListFixture = `<?xml version="1.0" encoding="UTF-8"?>
<collection xmlns="http://www.arin.net/regrws/core/v1">
  <roaSpec xmlns="http://www.arin.net/regrws/rpki/v1">
    <asNumber>64496</asNumber>
    <name>IANA-RSVD</name>
    <notValidAfter>2020-12-13T00:00:00-05:00</notValidAfter>
    <notValidBefore>2019-12-14T00:00:00-05:00</notValidBefore>
    <resources>
      <cidrLength>32</cidrLength>
      <endAddress>2001:db8:ffff:ffff:ffff:ffff:ffff:ffff</endAddress>
      <ipVersion>6</ipVersion>
      <maxLength>48</maxLength>
      <startAddress>2001:db8:0:0:0:0:0:0</startAddress>
      <autoLinked>true</autoLinked>
    </resources>
    <roaHandle>58bc1674f7784054ba743b9f5c23885b</roaHandle>
  </roaSpec>
  <roaSpec xmlns="http://www.arin.net/regrws/rpki/v1">
    <asNumber>64497</asNumber>
    <name>second</name>
    <resources>
      <cidrLength>24</cidrLength>
      <endAddress>192.0.2.255</endAddress>
      <ipVersion>4</ipVersion>
      <startAddress>192.0.2.0</startAddress>
      <autoLinked>false</autoLinked>
    </resources>
    <roaHandle>aa00</roaHandle>
  </roaSpec>
</collection>`

// the verbatim single-roa response shape from the arin docs
const roaBareFixture = `<roaSpec xmlns="http://www.arin.net/regrws/rpki/v1">
  <asNumber>64496</asNumber>
  <name>IANA-RSVD</name>
  <notValidAfter>2020-12-13T00:00:00-05:00</notValidAfter>
  <notValidBefore>2019-12-14T00:00:00-05:00</notValidBefore>
  <resources>
    <cidrLength>32</cidrLength>
    <endAddress>2001:db8:ffff:ffff:ffff:ffff:ffff:ffff</endAddress>
    <ipVersion>6</ipVersion>
    <maxLength>48</maxLength>
    <startAddress>2001:db8:0:0:0:0:0:0</startAddress>
    <autoLinked>true</autoLinked>
  </resources>
  <roaHandle>58bc1674f7784054ba743b9f5c23885b</roaHandle>
</roaSpec>`

const aspaListFixture = `<?xml version="1.0" encoding="UTF-8"?>
<collection xmlns="http://www.arin.net/regrws/core/v1">
  <aspa>
    <customerAsId>64496</customerAsId>
    <providerAsIds>
      <providerAsId>64497</providerAsId>
      <providerAsId>64498</providerAsId>
    </providerAsIds>
  </aspa>
</collection>`

const errorFixture = `<error xmlns="http://www.arin.net/regrws/core/v1">
  <message>Object not found</message>
  <code>E_OBJECT_NOT_FOUND</code>
  <components>
    <component>
      <name>roaHandle</name>
      <message>no such handle</message>
    </component>
  </components>
  <additionalInfo>
    <message>extra</message>
  </additionalInfo>
</error>`

func serve(t *testing.T, status int, payload string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		if _, err := io.WriteString(w, payload); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestROAsDecodesWrappedList(t *testing.T) {
	c := serve(t, http.StatusOK, roaListFixture)
	roas, err := c.ROAs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roas) != 2 {
		t.Fatalf("len = %d", len(roas))
	}
	first := roas[0]
	if first.Handle != "58bc1674f7784054ba743b9f5c23885b" || first.ASNumber != 64496 || first.Name != "IANA-RSVD" {
		t.Fatalf("first = %+v", first)
	}
	if len(first.Resources) != 1 {
		t.Fatalf("resources = %+v", first.Resources)
	}
	r := first.Resources[0]
	if r.StartAddress != "2001:db8:0:0:0:0:0:0" || r.CIDRLength != 32 || r.IPVersion != 6 || !r.AutoLinked {
		t.Fatalf("resource = %+v", r)
	}
	if r.MaxLength == nil || *r.MaxLength != 48 {
		t.Fatalf("maxLength = %v", r.MaxLength)
	}
	if roas[1].Resources[0].MaxLength != nil {
		t.Fatalf("absent maxLength decoded as %v", *roas[1].Resources[0].MaxLength)
	}
}

func TestROAsDecodesBareRoot(t *testing.T) {
	c := serve(t, http.StatusOK, roaBareFixture)
	roas, err := c.ROAs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roas) != 1 || roas[0].ASNumber != 64496 {
		t.Fatalf("roas = %+v", roas)
	}
}

func TestASPAsDecodes(t *testing.T) {
	c := serve(t, http.StatusOK, aspaListFixture)
	aspas, err := c.ASPAs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(aspas) != 1 || aspas[0].CustomerASID != 64496 {
		t.Fatalf("aspas = %+v", aspas)
	}
	if !reflect.DeepEqual(aspas[0].ProviderASIDs, []int64{64497, 64498}) {
		t.Fatalf("providers = %v", aspas[0].ProviderASIDs)
	}
}

func TestErrorPayload(t *testing.T) {
	c := serve(t, http.StatusNotFound, errorFixture)
	_, err := c.ROAs(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound = false for %v", err)
	}
	msg := err.Error()
	for _, want := range []string{"E_OBJECT_NOT_FOUND", "http 404", "roaHandle", "no such handle", "extra"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestTransportErrorOmitsKey(t *testing.T) {
	srv := httptest.NewServer(nil)
	srv.Close()
	c, err := New(srv.URL, "SECRETKEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ROAs(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "SECRETKEY") {
		t.Fatalf("error leaks the api key: %q", err)
	}
	if !strings.Contains(err.Error(), "GET /rest/roa/ORG") {
		t.Fatalf("error lost method and path: %q", err)
	}
}

func TestTransportErrorKeepsCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "SECRETKEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.ROAs(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cause chain broken: %v", err)
	}
	if strings.Contains(err.Error(), "SECRETKEY") {
		t.Fatalf("error leaks the api key: %q", err)
	}
}

func TestErrorUnparseable(t *testing.T) {
	c := serve(t, http.StatusInternalServerError, "boom")
	_, err := c.ROAs(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if IsNotFound(err) {
		t.Fatal("IsNotFound = true")
	}
	if !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("error = %q", err)
	}
}
