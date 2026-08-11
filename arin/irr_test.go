package arin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// fixtures mirror documents captured live from ote on 2026-08-11,
// with values replaced by documentation ranges
const irrRouteFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><route xmlns="http://www.arin.net/regrws/core/v1"><creationDate>2025-08-26T11:45:28Z</creationDate><description><line number="0">example line one</line><line number="1">example line two</line></description><lastModifiedDate>2025-08-26T11:48:54Z</lastModifiedDate><orgHandle>EXAMPLE</orgHandle><pocLinks><pocLinkRef description="Routing" function="R" handle="EXAMPLER-ARIN"/><pocLinkRef description="Tech" function="T" handle="EXAMPLET-ARIN"/><pocLinkRef description="Admin" function="AD" handle="EXAMPLEA-ARIN"/></pocLinks><remarks><line number="0">managed remark</line></remarks><source>ARIN</source><autoLinkedRoaHandle>42cfa7eb972e42c289a63bbc8d31ba80</autoLinkedRoaHandle><memberOf><routeSetRef name="RS-EXAMPLE"/></memberOf><netHandle>NET-192-0-2-0-1</netHandle><originAS>AS64496</originAS><prefix>192.000.002.000/24</prefix><version>4</version></route>`

const irrAutNumFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><autnum xmlns="http://www.arin.net/regrws/core/v1"><creationDate>2026-04-10T12:26:18Z</creationDate><description><line number="0">example</line></description><lastModifiedDate>2026-04-10T12:26:18Z</lastModifiedDate><orgHandle>EXAMPLE</orgHandle><pocLinks><pocLinkRef description="Routing" function="R" handle="EXAMPLER-ARIN"/><pocLinkRef description="Tech" function="T" handle="EXAMPLET-ARIN"/><pocLinkRef description="Admin" function="AD" handle="EXAMPLEA-ARIN"/></pocLinks><source>ARIN</source><asName>EXAMPLE</asName><asNumber>AS64496</asNumber><memberOf name="AS64496:AS-EXAMPLE"/><memberOf name="AS64496:AS-OWN"/><mpDefault><line number="0">to AS64496:AS-TRANSIT networks ANY</line></mpDefault><mpExport><line number="0">afi any to AS64496:AS-UP announce AS64496:AS-TRANSIT</line></mpExport><mpImport><line number="0">afi any from AS64496:AS-UP accept ANY</line></mpImport></autnum>`

const irrASSetFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><asSet xmlns="http://www.arin.net/regrws/core/v1"><creationDate>2025-08-26T11:58:53Z</creationDate><description><line number="0">example</line></description><lastModifiedDate>2025-08-26T12:04:04Z</lastModifiedDate><orgHandle>EXAMPLE</orgHandle><pocLinks><pocLinkRef description="Tech" function="T" handle="EXAMPLET-ARIN"/><pocLinkRef description="Admin" function="AD" handle="EXAMPLEA-ARIN"/></pocLinks><source>ARIN</source><members><member name="AS64496:AS-OWN"/></members><membersByRef><memberByRef name="MNT-EXAMPLE"/></membersByRef><name>AS64496:AS-EXAMPLE</name></asSet>`

const irrRouteSetFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><routeSet xmlns="http://www.arin.net/regrws/core/v1"><creationDate>2024-12-05T02:20:52Z</creationDate><description><line number="0">example</line></description><lastModifiedDate>2025-08-25T17:05:40Z</lastModifiedDate><orgHandle>EXAMPLE</orgHandle><pocLinks><pocLinkRef description="Admin" function="AD" handle="EXAMPLEA-ARIN"/><pocLinkRef description="Tech" function="T" handle="EXAMPLET-ARIN"/></pocLinks><source>ARIN</source><membersByRef><memberByRef name="MNT-EXAMPLE"/></membersByRef><name>RS-EXAMPLE</name><mpMembers><mpMember name="192.0.2.0/24"/><mpMember name="2001:db8::/32"/></mpMembers></routeSet>`

func TestIRRRouteDecodes(t *testing.T) {
	c := serve(t, http.StatusOK, irrRouteFixture)
	r, err := c.Route(context.Background(), "192.0.2.0/24", 64496)
	if err != nil {
		t.Fatal(err)
	}
	if r.Prefix != "192.000.002.000/24" || r.OriginAS != "AS64496" || r.Version != 4 {
		t.Fatalf("route = %+v", r)
	}
	if got := r.Description.Strings(); !reflect.DeepEqual(got, []string{"example line one", "example line two"}) {
		t.Fatalf("description = %v", got)
	}
	if got := r.Remarks.Strings(); !reflect.DeepEqual(got, []string{"managed remark"}) {
		t.Fatalf("remarks = %v", got)
	}
	if len(r.PocLinks) != 3 || r.PocLinks[0].Function != "R" || r.PocLinks[0].Handle != "EXAMPLER-ARIN" {
		t.Fatalf("pocLinks = %+v", r.PocLinks)
	}
	if r.AutoLinkedRoaHandle != "42cfa7eb972e42c289a63bbc8d31ba80" {
		t.Fatalf("autoLinkedRoaHandle = %q", r.AutoLinkedRoaHandle)
	}
	if len(r.MemberOf) != 1 || r.MemberOf[0].Name != "RS-EXAMPLE" {
		t.Fatalf("memberOf = %+v", r.MemberOf)
	}
	if r.NetHandle != "NET-192-0-2-0-1" || r.OrgHandle != "EXAMPLE" {
		t.Fatalf("route = %+v", r)
	}
}

func TestIRRAutNumDecodes(t *testing.T) {
	c := serve(t, http.StatusOK, irrAutNumFixture)
	a, err := c.AutNum(context.Background(), 64496)
	if err != nil {
		t.Fatal(err)
	}
	if a.ASNumber != "AS64496" || a.ASName != "EXAMPLE" {
		t.Fatalf("autnum = %+v", a)
	}
	if len(a.MemberOf) != 2 || a.MemberOf[0].Name != "AS64496:AS-EXAMPLE" {
		t.Fatalf("memberOf = %+v", a.MemberOf)
	}
	if got := a.MPImports.Strings(); !reflect.DeepEqual(got, []string{"afi any from AS64496:AS-UP accept ANY"}) {
		t.Fatalf("mpImport = %v", got)
	}
	if a.Imports != nil {
		t.Fatalf("imports = %+v", a.Imports)
	}
}

func TestIRRASSetDecodes(t *testing.T) {
	c := serve(t, http.StatusOK, irrASSetFixture)
	s, err := c.ASSet(context.Background(), "AS64496:AS-EXAMPLE")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "AS64496:AS-EXAMPLE" {
		t.Fatalf("asSet = %+v", s)
	}
	if len(s.Members) != 1 || s.Members[0].Name != "AS64496:AS-OWN" {
		t.Fatalf("members = %+v", s.Members)
	}
	if len(s.MembersByRef) != 1 || s.MembersByRef[0].Name != "MNT-EXAMPLE" {
		t.Fatalf("membersByRef = %+v", s.MembersByRef)
	}
}

func TestIRRRouteSetDecodes(t *testing.T) {
	c := serve(t, http.StatusOK, irrRouteSetFixture)
	s, err := c.RouteSet(context.Background(), "RS-EXAMPLE")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "RS-EXAMPLE" {
		t.Fatalf("routeSet = %+v", s)
	}
	if len(s.MPMembers) != 2 || s.MPMembers[1].Name != "2001:db8::/32" {
		t.Fatalf("mpMembers = %+v", s.MPMembers)
	}
}

// ote returns 404 with E_UNSPECIFIED for missing irr objects, so 404
// alone must count as missing, but not as the stricter IsNotFound
// that rpki call sites rely on
func TestIsMissingOn404Unspecified(t *testing.T) {
	c := serve(t, http.StatusNotFound, `<error xmlns="http://www.arin.net/regrws/core/v1"><additionalInfo><message>An error occurred while attempting to process your request.</message></additionalInfo><code>E_UNSPECIFIED</code><components/><message>An error occurred while attempting to process your request.</message></error>`)
	_, err := c.Route(context.Background(), "192.0.2.0/24", 64496)
	if err == nil || !IsMissing(err) {
		t.Fatalf("err = %v", err)
	}
	if IsNotFound(err) {
		t.Fatalf("IsNotFound = true for %v", err)
	}
}

func TestLinesRoundTrip(t *testing.T) {
	if MakeLines(nil) != nil {
		t.Fatal("MakeLines(nil) not nil")
	}
	l := MakeLines([]string{"a", "b"})
	if got := l.Strings(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("round trip = %v", got)
	}
	var nilLines *Lines
	if nilLines.Strings() != nil {
		t.Fatal("nil Strings() not nil")
	}
}

func TestIRRRouteCreateRequest(t *testing.T) {
	var method, path, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/xml")
		if _, err := io.WriteString(w, irrRouteFixture); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	in := IRRRoute{
		Prefix:      "192.0.2.0/24",
		OriginAS:    "AS64496",
		Description: MakeLines([]string{"example"}),
		PocLinks:    []PocLinkRef{{Function: "AD", Handle: "EXAMPLEA-ARIN"}},
		OrgHandle:   "ORG",
		Source:      "ARIN",
	}
	out, err := c.RouteCreate(context.Background(), "192.0.2.0/24", 64496, in)
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/rest/irr/route/192.0.2.0/24/AS64496" {
		t.Fatalf("%s %s", method, path)
	}
	if !strings.Contains(body, `<route xmlns="http://www.arin.net/regrws/core/v1">`) {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(body, `<line number="0">example</line>`) {
		t.Fatalf("body = %s", body)
	}
	if out.NetHandle == "" {
		t.Fatal("response not decoded")
	}
}

func TestIRRASSetCreateQuery(t *testing.T) {
	var path, rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		if _, err := io.WriteString(w, irrASSetFixture); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ASSetCreate(context.Background(), IRRASSet{Name: "AS64496:AS-EXAMPLE"}); err != nil {
		t.Fatal(err)
	}
	if path != "/rest/irr/as-set" {
		t.Fatalf("path = %q", path)
	}
	if !strings.Contains(rawQuery, "orgHandle=ORG") {
		t.Fatalf("query = %q", rawQuery)
	}
}

func TestIRRDeletePaths(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/xml")
		if _, err := io.WriteString(w, irrAutNumFixture); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AutNumDelete(context.Background(), 64496); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/rest/irr/aut-num/AS64496" {
		t.Fatalf("%s %s", method, path)
	}
}

func TestIRRRoutesList(t *testing.T) {
	c := serve(t, http.StatusOK, `<?xml version="1.0"?><collection xmlns="http://www.arin.net/regrws/core/v1">`+irrRouteFixture[55:]+`</collection>`)
	routes, err := c.Routes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].OriginAS != "AS64496" {
		t.Fatalf("routes = %+v", routes)
	}
}

func TestIRRRouteV6Path(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/xml")
		if _, err := io.WriteString(w, irrRouteFixture); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "KEY", "ORG")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Route(context.Background(), "2001:db8::/32", 64496); err != nil {
		t.Fatal(err)
	}
	if path != "/rest/irr/route/2001:db8::/32/AS64496" {
		t.Fatalf("path = %q", path)
	}
}
