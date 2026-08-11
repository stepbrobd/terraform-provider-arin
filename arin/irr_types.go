package arin

import (
	"encoding/xml"
	"slices"
)

// Lines is arin's numbered multi-line text container
type Lines struct {
	Lines []Line `xml:"line"`
}

type Line struct {
	Number int    `xml:"number,attr"`
	Text   string `xml:",chardata"`
}

// Strings returns the text lines in number order, nil for nil input
func (l *Lines) Strings() []string {
	if l == nil {
		return nil
	}
	sorted := slices.Clone(l.Lines)
	slices.SortStableFunc(sorted, func(a, b Line) int { return a.Number - b.Number })
	out := make([]string, 0, len(sorted))
	for _, ln := range sorted {
		out = append(out, ln.Text)
	}
	return out
}

// MakeLines builds a numbered container, nil for empty input
func MakeLines(ss []string) *Lines {
	if len(ss) == 0 {
		return nil
	}
	l := &Lines{}
	for i, s := range ss {
		l.Lines = append(l.Lines, Line{Number: i, Text: s})
	}
	return l
}

// PocLinkRef links a poc handle with a function
// functions are AD (admin), T (tech), R (routing)
type PocLinkRef struct {
	Description string `xml:"description,attr,omitempty"`
	Function    string `xml:"function,attr"`
	Handle      string `xml:"handle,attr"`
}

// NameRef is arin's name-attribute reference element
type NameRef struct {
	Name string `xml:"name,attr"`
}

// IRRRoute is the wire shape of an irr route object
// both address families use this root, distinguished by Version
type IRRRoute struct {
	XMLName             xml.Name     `xml:"http://www.arin.net/regrws/core/v1 route"`
	Created             string       `xml:"creationDate,omitempty"`
	Description         *Lines       `xml:"description,omitempty"`
	Modified            string       `xml:"lastModifiedDate,omitempty"`
	OrgHandle           string       `xml:"orgHandle,omitempty"`
	PocLinks            []PocLinkRef `xml:"pocLinks>pocLinkRef,omitempty"`
	Remarks             *Lines       `xml:"remarks,omitempty"`
	Source              string       `xml:"source,omitempty"`
	AutoLinkedRoaHandle string       `xml:"autoLinkedRoaHandle,omitempty"`
	MemberOf            []NameRef    `xml:"memberOf>routeSetRef,omitempty"`
	NetHandle           string       `xml:"netHandle,omitempty"`
	OriginAS            string       `xml:"originAS"`
	Prefix              string       `xml:"prefix"`
	Version             int64        `xml:"version,omitempty"`
}

// IRRAutNum is the wire shape of an irr aut-num object
type IRRAutNum struct {
	XMLName     xml.Name     `xml:"http://www.arin.net/regrws/core/v1 autnum"`
	Created     string       `xml:"creationDate,omitempty"`
	Description *Lines       `xml:"description,omitempty"`
	Modified    string       `xml:"lastModifiedDate,omitempty"`
	OrgHandle   string       `xml:"orgHandle,omitempty"`
	PocLinks    []PocLinkRef `xml:"pocLinks>pocLinkRef,omitempty"`
	Remarks     *Lines       `xml:"remarks,omitempty"`
	Source      string       `xml:"source,omitempty"`
	ASName      string       `xml:"asName"`
	ASNumber    string       `xml:"asNumber"`
	MemberOf    []NameRef    `xml:"memberOf,omitempty"`
	Imports     *Lines       `xml:"import,omitempty"`
	Exports     *Lines       `xml:"export,omitempty"`
	Defaults    *Lines       `xml:"default,omitempty"`
	MPImports   *Lines       `xml:"mpImport,omitempty"`
	MPExports   *Lines       `xml:"mpExport,omitempty"`
	MPDefaults  *Lines       `xml:"mpDefault,omitempty"`
}

// IRRASSet is the wire shape of an irr as-set object
type IRRASSet struct {
	XMLName      xml.Name     `xml:"http://www.arin.net/regrws/core/v1 asSet"`
	Created      string       `xml:"creationDate,omitempty"`
	Description  *Lines       `xml:"description,omitempty"`
	Modified     string       `xml:"lastModifiedDate,omitempty"`
	OrgHandle    string       `xml:"orgHandle,omitempty"`
	PocLinks     []PocLinkRef `xml:"pocLinks>pocLinkRef,omitempty"`
	Remarks      *Lines       `xml:"remarks,omitempty"`
	Source       string       `xml:"source,omitempty"`
	Name         string       `xml:"name"`
	Members      []NameRef    `xml:"members>member,omitempty"`
	MembersByRef []NameRef    `xml:"membersByRef>memberByRef,omitempty"`
}

// IRRRouteSet is the wire shape of an irr route-set object
type IRRRouteSet struct {
	XMLName      xml.Name     `xml:"http://www.arin.net/regrws/core/v1 routeSet"`
	Created      string       `xml:"creationDate,omitempty"`
	Description  *Lines       `xml:"description,omitempty"`
	Modified     string       `xml:"lastModifiedDate,omitempty"`
	OrgHandle    string       `xml:"orgHandle,omitempty"`
	PocLinks     []PocLinkRef `xml:"pocLinks>pocLinkRef,omitempty"`
	Remarks      *Lines       `xml:"remarks,omitempty"`
	Source       string       `xml:"source,omitempty"`
	Name         string       `xml:"name"`
	Members      []NameRef    `xml:"members>member,omitempty"`
	MPMembers    []NameRef    `xml:"mpMembers>mpMember,omitempty"`
	MembersByRef []NameRef    `xml:"membersByRef>memberByRef,omitempty"`
}
