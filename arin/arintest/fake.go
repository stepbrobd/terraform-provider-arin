// Package arintest is an in-memory fake of the arin rpki restful api
package arintest

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sort"
	"sync"
	"testing"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// Server holds fake org state behind the three rpki endpoints
type Server struct {
	*httptest.Server

	mu    sync.Mutex
	key   string
	org   string
	seq   int
	roas  map[string]arin.ROASpec
	aspas map[int64]arin.ASPA
}

// New starts a fake bound to one api key and org handle
// the server closes with the test
func New(t *testing.T, key, org string) *Server {
	s := &Server{
		key:   key,
		org:   org,
		roas:  map[string]arin.ROASpec{},
		aspas: map[int64]arin.ASPA{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) fail(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	b, err := xml.Marshal(&arin.Error{Code: code, Message: msg})
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// ote ignores the authorization header, so the fake authenticates
	// the way ote actually does
	if r.URL.Query().Get("apikey") != s.key {
		s.fail(w, http.StatusUnauthorized, arin.CodeAuthentication, "bad api key")
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/rest/rpki/"+s.org:
		s.transact(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/rest/roa/"+s.org:
		s.listROAs(w)
	case r.Method == http.MethodGet && r.URL.Path == "/rest/aspa/"+s.org:
		s.listASPAs(w)
	default:
		s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, r.Method+" "+r.URL.Path)
	}
}

func (s *Server) transact(w http.ResponseWriter, r *http.Request) {
	var tx arin.Transaction
	if err := xml.NewDecoder(r.Body).Decode(&tx); err != nil {
		s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
		return
	}
	// validate everything before mutating so the transaction stays
	// all-or-nothing like the real api
	if tx.ROADeletes != nil {
		for _, h := range tx.ROADeletes.Handles {
			if _, ok := s.roas[h.Handle]; !ok {
				s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "roa "+h.Handle)
				return
			}
		}
	}
	if tx.ASPADeletes != nil {
		for _, id := range tx.ASPADeletes.CustomerASIDs {
			if _, ok := s.aspas[id]; !ok {
				s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, fmt.Sprintf("aspa %d", id))
				return
			}
		}
	}
	var adds []arin.ROASpec
	if tx.ROAAdds != nil {
		for _, spec := range tx.ROAAdds.Specs {
			built, err := s.build(spec)
			if err != nil {
				s.fail(w, http.StatusBadRequest, arin.CodeEntityValidation, err.Error())
				return
			}
			adds = append(adds, built)
		}
	}
	if tx.ROADeletes != nil {
		for _, h := range tx.ROADeletes.Handles {
			delete(s.roas, h.Handle)
		}
	}
	if tx.ASPADeletes != nil {
		for _, id := range tx.ASPADeletes.CustomerASIDs {
			delete(s.aspas, id)
		}
	}
	for _, a := range adds {
		s.roas[a.Handle] = a
	}
	if tx.ASPAAdds != nil {
		for _, a := range tx.ASPAAdds.ASPAs {
			a.XMLName = xml.Name{}
			s.aspas[a.CustomerASID] = a
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) build(spec arin.ROASpecRequest) (arin.ROASpec, error) {
	s.seq++
	out := arin.ROASpec{
		Handle:         fmt.Sprintf("%032x", s.seq),
		ASNumber:       spec.ASNumber,
		Name:           spec.Name,
		AutoRenewed:    true,
		NotValidBefore: "2026-01-01T00:00:00-05:00",
		NotValidAfter:  "2027-01-01T00:00:00-05:00",
	}
	for _, rr := range spec.Resources {
		addr, err := netip.ParseAddr(rr.StartAddress)
		if err != nil {
			return out, fmt.Errorf("start address %q: %w", rr.StartAddress, err)
		}
		p, err := addr.Prefix(int(rr.CIDRLength))
		if err != nil {
			return out, fmt.Errorf("cidr length %d: %w", rr.CIDRLength, err)
		}
		ver := int64(4)
		if addr.Is6() {
			ver = 6
		}
		ml := rr.MaxLength
		if ml == nil {
			// arin defaults max length to the cidr length when unset
			v := rr.CIDRLength
			ml = &v
		}
		start := rr.StartAddress
		end := lastAddr(p).String()
		if addr.Is4() {
			start = pad4(addr)
			end = pad4(lastAddr(p))
		}
		out.Resources = append(out.Resources, arin.ROAResource{
			StartAddress: start,
			EndAddress:   end,
			CIDRLength:   rr.CIDRLength,
			MaxLength:    ml,
			IPVersion:    ver,
			AutoLinked:   spec.AutoLink,
		})
	}
	// serve resources in a canonical order unrelated to request order
	sort.Slice(out.Resources, func(i, j int) bool {
		return out.Resources[i].StartAddress < out.Resources[j].StartAddress
	})
	return out, nil
}

// pad4 mirrors ote's zero padded dotted quad rendering
func pad4(a netip.Addr) string {
	b := a.As4()
	return fmt.Sprintf("%03d.%03d.%03d.%03d", b[0], b[1], b[2], b[3])
}

// lastAddr returns the highest address in p
func lastAddr(p netip.Prefix) netip.Addr {
	if p.Addr().Is4() {
		b := p.Masked().Addr().As4()
		for i := p.Bits(); i < 32; i++ {
			b[i/8] |= 1 << (7 - i%8)
		}
		return netip.AddrFrom4(b)
	}
	b := p.Masked().Addr().As16()
	for i := p.Bits(); i < 128; i++ {
		b[i/8] |= 1 << (7 - i%8)
	}
	return netip.AddrFrom16(b)
}

func (s *Server) listROAs(w http.ResponseWriter) {
	type collection struct {
		XMLName xml.Name       `xml:"http://www.arin.net/regrws/core/v1 collection"`
		Items   []arin.ROASpec `xml:"roaSpec"`
	}
	var c collection
	for _, v := range s.roas {
		v.XMLName = xml.Name{}
		c.Items = append(c.Items, v)
	}
	sort.Slice(c.Items, func(i, j int) bool { return c.Items[i].Handle < c.Items[j].Handle })
	s.write(w, c)
}

func (s *Server) listASPAs(w http.ResponseWriter) {
	type collection struct {
		XMLName xml.Name    `xml:"http://www.arin.net/regrws/core/v1 collection"`
		Items   []arin.ASPA `xml:"aspa"`
	}
	var c collection
	for _, v := range s.aspas {
		c.Items = append(c.Items, v)
	}
	sort.Slice(c.Items, func(i, j int) bool { return c.Items[i].CustomerASID < c.Items[j].CustomerASID })
	s.write(w, c)
}

func (s *Server) write(w http.ResponseWriter, v any) {
	b, err := xml.Marshal(v)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
}
