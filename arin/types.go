package arin

import "encoding/xml"

// Transaction is the request body for POST /rest/rpki/ORGHANDLE
// deletes apply before adds and the whole transaction is atomic
type Transaction struct {
	XMLName     xml.Name     `xml:"http://www.arin.net/regrws/rpki/v1 rpkiTransaction"`
	ROADeletes  *ROADeletes  `xml:"roaSpecDelete,omitempty"`
	ROAAdds     *ROAAdds     `xml:"roaSpecAdd,omitempty"`
	ASPADeletes *ASPADeletes `xml:"aspaDelete,omitempty"`
	ASPAAdds    *ASPAAdds    `xml:"aspaAdd,omitempty"`
}

type ROADeletes struct {
	Handles []ROAHandleRef `xml:"roaHandle"`
}

// ROAHandleRef names an existing roa in a delete
// the docs show a lowercase autolink attribute but the live schema
// requires autoLink, verified against ote on 2026-08-11
type ROAHandleRef struct {
	AutoLink bool   `xml:"autoLink,attr"`
	Handle   string `xml:",chardata"`
}

type ROAAdds struct {
	Specs []ROASpecRequest `xml:"roaSpec"`
}

// ROASpecRequest is the client-supplied shape of a roa
type ROASpecRequest struct {
	AutoLink  bool                 `xml:"autoLink"`
	ASNumber  int64                `xml:"asNumber"`
	Name      string               `xml:"name"`
	Resources []ROAResourceRequest `xml:"resources>roaSpecResource"`
}

type ROAResourceRequest struct {
	StartAddress string `xml:"startAddress"`
	CIDRLength   int64  `xml:"cidrLength"`
	MaxLength    *int64 `xml:"maxLength,omitempty"`
}

// ROASpec is the server-reported shape of a roa
// responses flatten each resource into a repeated resources element
type ROASpec struct {
	XMLName        xml.Name      `xml:"roaSpec"`
	Handle         string        `xml:"roaHandle"`
	ASNumber       int64         `xml:"asNumber"`
	Name           string        `xml:"name"`
	AutoRenewed    bool          `xml:"autoRenewed"`
	NotValidBefore string        `xml:"notValidBefore"`
	NotValidAfter  string        `xml:"notValidAfter"`
	Resources      []ROAResource `xml:"resources"`
}

type ROAResource struct {
	StartAddress string `xml:"startAddress"`
	EndAddress   string `xml:"endAddress"`
	CIDRLength   int64  `xml:"cidrLength"`
	MaxLength    *int64 `xml:"maxLength"`
	IPVersion    int64  `xml:"ipVersion"`
	AutoLinked   bool   `xml:"autoLinked"`
}

type ASPADeletes struct {
	CustomerASIDs []int64 `xml:"customerAsId"`
}

type ASPAAdds struct {
	ASPAs []ASPA `xml:"aspa"`
}

// ASPA is both the request and response shape of an aspa
type ASPA struct {
	XMLName       xml.Name `xml:"aspa"`
	CustomerASID  int64    `xml:"customerAsId"`
	ProviderASIDs []int64  `xml:"providerAsIds>providerAsId"`
}
