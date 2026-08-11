package arin

import "encoding/xml"

// registry object shapes for nets, delegations, orgs, and pocs
// container fields use pointer wrappers so empty containers round-trip
// byte-identically on echo puts

// PocLinks is the pocLinks container element
type PocLinks struct {
	Refs []PocLinkRef `xml:"pocLinkRef"`
}

// OriginASes is the originASes container element
type OriginASes struct {
	ASes []string `xml:"originAS"`
}

// NetBlock is one address block of a net
type NetBlock struct {
	CIDRLength   int64  `xml:"cidrLength"`
	Description  string `xml:"description,omitempty"`
	EndAddress   string `xml:"endAddress"`
	StartAddress string `xml:"startAddress"`
	Type         string `xml:"type"`
}

// NetBlocks is the netBlocks container element
type NetBlocks struct {
	Blocks []NetBlock `xml:"netBlock"`
}

// Net is the wire shape of a registry net
type Net struct {
	XMLName          xml.Name    `xml:"http://www.arin.net/regrws/core/v1 net"`
	PocLinks         *PocLinks   `xml:"pocLinks,omitempty"`
	Comment          *Lines      `xml:"comment,omitempty"`
	NetBlocks        *NetBlocks  `xml:"netBlocks,omitempty"`
	Handle           string      `xml:"handle,omitempty"`
	NetName          string      `xml:"netName,omitempty"`
	OrgHandle        string      `xml:"orgHandle,omitempty"`
	OriginASes       *OriginASes `xml:"originASes,omitempty"`
	ParentNetHandle  string      `xml:"parentNetHandle,omitempty"`
	RegistrationDate string      `xml:"registrationDate,omitempty"`
	Version          int64       `xml:"version,omitempty"`
}

// NamedValue is arin's numbered element with a decorative name attribute
// the chardata stays a string, callers parse the number
type NamedValue struct {
	Name  string `xml:"name,attr,omitempty"`
	Value string `xml:",chardata"`
}

// DelegationKey is one ds record of a delegation
type DelegationKey struct {
	Algorithm  NamedValue `xml:"algorithm"`
	Digest     string     `xml:"digest"`
	DigestType NamedValue `xml:"digestType"`
	KeyTag     int64      `xml:"keyTag"`
	TTL        int64      `xml:"http://www.arin.net/regrws/ttl/v1 ttl"`
}

// DelegationKeys is the delegationKeys container element
type DelegationKeys struct {
	Keys []DelegationKey `xml:"delegationKey"`
}

// Delegation is the wire shape of a reverse dns delegation
type Delegation struct {
	XMLName     xml.Name        `xml:"http://www.arin.net/regrws/core/v1 delegation"`
	Keys        *DelegationKeys `xml:"delegationKeys,omitempty"`
	Name        string          `xml:"name"`
	Nameservers []string        `xml:"nameservers>nameserver,omitempty"`
}

// ISO3166One is the iso3166-1 country element
type ISO3166One struct {
	Code2 string `xml:"code2"`
	Code3 string `xml:"code3,omitempty"`
	E164  string `xml:"e164,omitempty"`
	Name  string `xml:"name,omitempty"`
}

// Org is the wire shape of a registry org
type Org struct {
	XMLName             xml.Name   `xml:"http://www.arin.net/regrws/core/v1 org"`
	PocLinks            *PocLinks  `xml:"pocLinks,omitempty"`
	AcceptReassignments bool       `xml:"acceptReassignments,omitempty"`
	City                string     `xml:"city,omitempty"`
	ISO3166One          ISO3166One `xml:"iso3166-1"`
	Comment             *Lines     `xml:"comment,omitempty"`
	StreetAddress       *Lines     `xml:"streetAddress,omitempty"`
	Handle              string     `xml:"handle,omitempty"`
	OrgName             string     `xml:"orgName,omitempty"`
	DBAName             string     `xml:"dbaName,omitempty"`
	PostalCode          string     `xml:"postalCode,omitempty"`
	RegistrationDate    string     `xml:"registrationDate,omitempty"`
	ISO3166Two          string     `xml:"iso3166-2,omitempty"`
	TaxID               string     `xml:"taxId,omitempty"`
}

// PhoneType is the typed phone classifier
type PhoneType struct {
	Code        string `xml:"code"`
	Description string `xml:"description,omitempty"`
}

// Phone is one poc phone number
type Phone struct {
	Number string    `xml:"number"`
	Type   PhoneType `xml:"type"`
}

// Poc is the wire shape of a registry point of contact
type Poc struct {
	XMLName          xml.Name   `xml:"http://www.arin.net/regrws/core/v1 poc"`
	City             string     `xml:"city,omitempty"`
	CompanyName      string     `xml:"companyName,omitempty"`
	ContactType      string     `xml:"contactType,omitempty"`
	ISO3166One       ISO3166One `xml:"iso3166-1"`
	Emails           []string   `xml:"emails>email,omitempty"`
	FirstName        string     `xml:"firstName,omitempty"`
	LastName         string     `xml:"lastName,omitempty"`
	Comment          *Lines     `xml:"comment,omitempty"`
	StreetAddress    *Lines     `xml:"streetAddress,omitempty"`
	Phones           []Phone    `xml:"phones>phone,omitempty"`
	Handle           string     `xml:"handle,omitempty"`
	PostalCode       string     `xml:"postalCode,omitempty"`
	RegistrationDate string     `xml:"registrationDate,omitempty"`
	ISO3166Two       string     `xml:"iso3166-2,omitempty"`
}
