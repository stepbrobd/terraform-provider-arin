package arintest

import (
	"encoding/xml"
	"net/http"
	"reflect"
	"strings"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// registry endpoints: net and delegation get/put, org and poc get
// missing objects fail with E_OBJECT_NOT_FOUND, matching the live
// registry endpoints rather than the bare-404 irr behavior

// InjectNet seeds a net keyed by handle
func (s *Server) InjectNet(n arin.Net) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nets[n.Handle] = n
}

// InjectDelegation seeds a delegation keyed by its undotted name
func (s *Server) InjectDelegation(d arin.Delegation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasSuffix(d.Name, ".") {
		d.Name += "."
	}
	s.delegations[strings.TrimSuffix(d.Name, ".")] = d
}

// InjectOrg seeds an org keyed by handle
func (s *Server) InjectOrg(o arin.Org) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orgs[o.Handle] = o
}

// InjectPoc seeds a poc keyed by handle
func (s *Server) InjectPoc(p arin.Poc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pocs[p.Handle] = p
}

func (s *Server) registry(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/rest/")
	kind, id, ok := strings.Cut(rest, "/")
	if !ok || id == "" {
		s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, r.URL.Path)
		return
	}
	switch kind {
	case "net":
		s.net(w, r, id)
	case "delegation":
		s.delegation(w, r, strings.TrimSuffix(id, "."))
	case "org":
		if r.Method != http.MethodGet {
			s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
			return
		}
		o, ok := s.orgs[id]
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "org "+id)
			return
		}
		s.write(w, o)
	case "poc":
		if r.Method != http.MethodGet {
			s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
			return
		}
		p, ok := s.pocs[id]
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "poc "+id)
			return
		}
		s.write(w, p)
	default:
		s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, r.URL.Path)
	}
}

func (s *Server) net(w http.ResponseWriter, r *http.Request, handle string) {
	existing, ok := s.nets[handle]
	switch r.Method {
	case http.MethodGet:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "net "+handle)
			return
		}
		s.write(w, existing)
	case http.MethodPut:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "net "+handle)
			return
		}
		var in arin.Net
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		// net blocks are server assigned and immutable
		if in.NetBlocks != nil && !reflect.DeepEqual(in.NetBlocks, existing.NetBlocks) {
			s.fail(w, http.StatusBadRequest, arin.CodeEntityValidation, "netBlocks: This element cannot be modified.")
			return
		}
		out := existing
		out.NetName = in.NetName
		out.Comment = in.Comment
		s.nets[handle] = out
		s.write(w, out)
	default:
		s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
	}
}

func (s *Server) delegation(w http.ResponseWriter, r *http.Request, name string) {
	existing, ok := s.delegations[name]
	switch r.Method {
	case http.MethodGet:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "delegation "+name)
			return
		}
		s.write(w, existing)
	case http.MethodPut:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeObjectNotFound, "delegation "+name)
			return
		}
		var in arin.Delegation
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		out := existing
		out.Nameservers = in.Nameservers
		out.Keys = in.Keys
		s.delegations[name] = out
		s.write(w, out)
	default:
		s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
	}
}
