package arin

import (
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

// live ote captures from 2026-08-12, embedded verbatim

const netFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><net xmlns="http://www.arin.net/regrws/core/v1" xmlns:ns2="http://www.arin.net/regrws/messages/v1"><pocLinks/><comment><line number="0">https://stepbrobd.com</line><line number="1">Geofeed https://stepbrobd.com/geofeed.csv</line></comment><netBlocks><netBlock><cidrLength>24</cidrLength><description>Direct Allocation</description><endAddress>023.161.104.255</endAddress><startAddress>023.161.104.000</startAddress><type>DA</type></netBlock></netBlocks><handle>NET-23-161-104-0-1</handle><netName>STEPBROBD</netName><orgHandle>STEPB</orgHandle><originASes/><parentNetHandle>NET-23-0-0-0-0</parentNetHandle><registrationDate>2023-08-31T17:50:06-04:00</registrationDate><version>4</version></net>`

const delegationFixture = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><delegation xmlns="http://www.arin.net/regrws/core/v1" xmlns:ns2="http://www.arin.net/regrws/ttl/v1"><delegationKeys><delegationKey><algorithm name="ECDSA Curve P-256 with SHA-256">13</algorithm><digest>2935C3B989F1B43F47CCBE08DC45D0F827D359E15E8CD1EA1EC2D8398022A1D8</digest><digestType name="SHA-256">2</digestType><keyTag>2371</keyTag><ns2:ttl>3600</ns2:ttl></delegationKey></delegationKeys><name>104.161.23.in-addr.arpa.</name><nameservers><nameserver>NS.YSUN.CO</nameserver><nameserver>NS.YSUN.FR</nameserver><nameserver>NS.YSUN.JP</nameserver><nameserver>NS.YSUN.US</nameserver></nameservers></delegation>`

func TestNetDecodesLiveShape(t *testing.T) {
	var n Net
	if err := xml.Unmarshal([]byte(netFixture), &n); err != nil {
		t.Fatal(err)
	}
	if n.Handle != "NET-23-161-104-0-1" || n.NetName != "STEPBROBD" || n.OrgHandle != "STEPB" {
		t.Fatalf("net = %+v", n)
	}
	if n.ParentNetHandle != "NET-23-0-0-0-0" || n.Version != 4 {
		t.Fatalf("net = %+v", n)
	}
	if got := n.Comment.Strings(); len(got) != 2 || got[1] != "Geofeed https://stepbrobd.com/geofeed.csv" {
		t.Fatalf("comment = %v", got)
	}
	if n.NetBlocks == nil || len(n.NetBlocks.Blocks) != 1 {
		t.Fatalf("netBlocks = %+v", n.NetBlocks)
	}
	b := n.NetBlocks.Blocks[0]
	if b.StartAddress != "023.161.104.000" || b.CIDRLength != 24 || b.Type != "DA" {
		t.Fatalf("netBlock = %+v", b)
	}
	// empty containers stay present as containers
	if n.PocLinks == nil || n.OriginASes == nil {
		t.Fatalf("containers lost: pocLinks=%v originASes=%v", n.PocLinks, n.OriginASes)
	}
}

func TestNetRoundTrips(t *testing.T) {
	var n Net
	if err := xml.Unmarshal([]byte(netFixture), &n); err != nil {
		t.Fatal(err)
	}
	out, err := xml.Marshal(&n)
	if err != nil {
		t.Fatal(err)
	}
	var back Net
	if err := xml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	back.XMLName = n.XMLName
	if !reflect.DeepEqual(n, back) {
		t.Fatalf("round trip mismatch\nfirst: %+v\nback:  %+v", n, back)
	}
}

func TestDelegationDecodesLiveShape(t *testing.T) {
	var d Delegation
	if err := xml.Unmarshal([]byte(delegationFixture), &d); err != nil {
		t.Fatal(err)
	}
	if d.Name != "104.161.23.in-addr.arpa." {
		t.Fatalf("name = %q", d.Name)
	}
	if len(d.Nameservers) != 4 || d.Nameservers[0] != "NS.YSUN.CO" {
		t.Fatalf("nameservers = %v", d.Nameservers)
	}
	if d.Keys == nil || len(d.Keys.Keys) != 1 {
		t.Fatalf("keys = %+v", d.Keys)
	}
	k := d.Keys.Keys[0]
	if k.Algorithm.Value != "13" || k.DigestType.Value != "2" || k.KeyTag != 2371 || k.TTL != 3600 {
		t.Fatalf("key = %+v", k)
	}
	if k.Digest != "2935C3B989F1B43F47CCBE08DC45D0F827D359E15E8CD1EA1EC2D8398022A1D8" {
		t.Fatalf("digest = %q", k.Digest)
	}
}

func TestDelegationRoundTrips(t *testing.T) {
	var d Delegation
	if err := xml.Unmarshal([]byte(delegationFixture), &d); err != nil {
		t.Fatal(err)
	}
	out, err := xml.Marshal(&d)
	if err != nil {
		t.Fatal(err)
	}
	// the ttl element must keep its namespace on the wire
	if !strings.Contains(string(out), "http://www.arin.net/regrws/ttl/v1") {
		t.Fatalf("marshal lost the ttl namespace: %s", out)
	}
	var back Delegation
	if err := xml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	back.XMLName = d.XMLName
	if !reflect.DeepEqual(d, back) {
		t.Fatalf("round trip mismatch\nfirst: %+v\nback:  %+v", d, back)
	}
}
