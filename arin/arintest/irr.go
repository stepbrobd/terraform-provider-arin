package arintest

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/netip"
	"sort"
	"strings"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

// routeKey canonicalizes the identity so padded and plain forms match
func routeKey(prefix, origin string) string {
	ip, length, _ := strings.Cut(prefix, "/")
	if a, err := netip.ParseAddr(unpad(ip)); err == nil {
		ip = a.String()
	}
	return ip + "/" + length + "|" + strings.ToUpper(origin)
}

// unpad strips per-octet leading zeros from dotted quads
func unpad(s string) string {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return s
	}
	for i, p := range parts {
		parts[i] = strings.TrimLeft(p, "0")
		if parts[i] == "" {
			parts[i] = "0"
		}
	}
	return strings.Join(parts, ".")
}

// InjectRoute seeds a route object directly, bypassing validation
// tests use it for server-owned states like auto-linked routes
func (s *Server) InjectRoute(r arin.IRRRoute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.irrRoutes[routeKey(r.Prefix, r.OriginAS)] = s.stampRoute(r)
}

func (s *Server) stamp() (created, modified string) {
	s.irrRev++
	return "2026-01-01T00:00:00Z", fmt.Sprintf("2026-01-01T00:00:%02dZ", s.irrRev%60)
}

func (s *Server) stampRoute(r arin.IRRRoute) arin.IRRRoute {
	created, modified := s.stamp()
	if r.Created == "" {
		r.Created = created
	}
	r.Modified = modified
	r.OrgHandle = s.org
	r.Source = "ARIN"
	ip, length, _ := strings.Cut(r.Prefix, "/")
	if a, err := netip.ParseAddr(unpad(ip)); err == nil {
		if a.Is4() {
			r.Version = 4
			r.Prefix = pad4(a) + "/" + length
			r.NetHandle = "NET-TEST-4"
		} else {
			r.Version = 6
			r.NetHandle = "NET-TEST-6"
		}
	}
	return r
}

func (s *Server) irr(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/rest/irr/")
	parts := strings.Split(rest, "/")
	switch {
	case parts[0] == "route" && len(parts) == 4:
		s.irrRoute(w, r, parts[1]+"/"+parts[2], parts[3])
	case parts[0] == "aut-num" && len(parts) == 2:
		s.irrAutNum(w, r, strings.ToUpper(parts[1]))
	case parts[0] == "as-set":
		s.irrSet(w, r, strings.Join(parts[1:], "/"), true)
	case parts[0] == "route-set":
		s.irrSet(w, r, strings.Join(parts[1:], "/"), false)
	case parts[0] == "org" && len(parts) == 3 && parts[1] == s.org && r.Method == http.MethodGet:
		s.irrList(w, parts[2])
	default:
		s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "irr "+r.URL.Path)
	}
}

func (s *Server) irrRoute(w http.ResponseWriter, r *http.Request, prefix, origin string) {
	key := routeKey(prefix, origin)
	existing, ok := s.irrRoutes[key]
	switch r.Method {
	case http.MethodGet:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route "+key)
			return
		}
		s.write(w, existing)
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost && ok {
			s.fail(w, http.StatusBadRequest, arin.CodeBadRequest, "duplicate route "+key)
			return
		}
		if r.Method == http.MethodPut && !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route "+key)
			return
		}
		var in arin.IRRRoute
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		out := s.stampRoute(in)
		if r.Method == http.MethodPut {
			out.Created = existing.Created
			out.AutoLinkedRoaHandle = existing.AutoLinkedRoaHandle
		}
		s.irrRoutes[key] = out
		s.write(w, out)
	case http.MethodDelete:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route "+key)
			return
		}
		delete(s.irrRoutes, key)
		s.write(w, existing)
	default:
		s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
	}
}

func (s *Server) irrAutNum(w http.ResponseWriter, r *http.Request, as string) {
	existing, ok := s.irrAutNums[as]
	switch r.Method {
	case http.MethodGet:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "aut-num "+as)
			return
		}
		s.write(w, existing)
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost && ok {
			s.fail(w, http.StatusBadRequest, arin.CodeBadRequest, "duplicate aut-num "+as)
			return
		}
		if r.Method == http.MethodPut && !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "aut-num "+as)
			return
		}
		var in arin.IRRAutNum
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		created, modified := s.stamp()
		if r.Method == http.MethodPut {
			created = existing.Created
		}
		in.Created = created
		in.Modified = modified
		in.OrgHandle = s.org
		in.Source = "ARIN"
		in.ASNumber = as
		s.irrAutNums[as] = in
		s.write(w, in)
	case http.MethodDelete:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "aut-num "+as)
			return
		}
		delete(s.irrAutNums, as)
		s.write(w, existing)
	default:
		s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
	}
}

// irrSet serves as-set and route-set, which differ only in payload type
func (s *Server) irrSet(w http.ResponseWriter, r *http.Request, name string, asSet bool) {
	if r.Method == http.MethodPost && name == "" {
		if r.URL.Query().Get("orgHandle") != s.org {
			s.fail(w, http.StatusBadRequest, arin.CodeBadRequest, "orgHandle required")
			return
		}
		if asSet {
			var in arin.IRRASSet
			if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
				s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
				return
			}
			if _, ok := s.irrASSets[in.Name]; ok {
				s.fail(w, http.StatusBadRequest, arin.CodeBadRequest, "duplicate as-set "+in.Name)
				return
			}
			in.Created, in.Modified = s.stamp()
			in.OrgHandle = s.org
			in.Source = "ARIN"
			s.irrASSets[in.Name] = in
			s.write(w, in)
			return
		}
		var in arin.IRRRouteSet
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		if _, ok := s.irrRouteSets[in.Name]; ok {
			s.fail(w, http.StatusBadRequest, arin.CodeBadRequest, "duplicate route-set "+in.Name)
			return
		}
		in.Created, in.Modified = s.stamp()
		in.OrgHandle = s.org
		in.Source = "ARIN"
		s.irrRouteSets[in.Name] = in
		s.write(w, in)
		return
	}
	if asSet {
		existing, ok := s.irrASSets[name]
		switch r.Method {
		case http.MethodGet:
			if !ok {
				s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "as-set "+name)
				return
			}
			s.write(w, existing)
		case http.MethodPut:
			if !ok {
				s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "as-set "+name)
				return
			}
			var in arin.IRRASSet
			if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
				s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
				return
			}
			in.Created = existing.Created
			_, in.Modified = s.stamp()
			in.OrgHandle = s.org
			in.Source = "ARIN"
			in.Name = name
			s.irrASSets[name] = in
			s.write(w, in)
		case http.MethodDelete:
			if !ok {
				s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "as-set "+name)
				return
			}
			delete(s.irrASSets, name)
			s.write(w, existing)
		default:
			s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
		}
		return
	}
	existing, ok := s.irrRouteSets[name]
	switch r.Method {
	case http.MethodGet:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route-set "+name)
			return
		}
		s.write(w, existing)
	case http.MethodPut:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route-set "+name)
			return
		}
		var in arin.IRRRouteSet
		if err := xml.NewDecoder(r.Body).Decode(&in); err != nil {
			s.fail(w, http.StatusBadRequest, arin.CodeSchemaValidation, err.Error())
			return
		}
		in.Created = existing.Created
		_, in.Modified = s.stamp()
		in.OrgHandle = s.org
		in.Source = "ARIN"
		in.Name = name
		s.irrRouteSets[name] = in
		s.write(w, in)
	case http.MethodDelete:
		if !ok {
			s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "route-set "+name)
			return
		}
		delete(s.irrRouteSets, name)
		s.write(w, existing)
	default:
		s.fail(w, http.StatusMethodNotAllowed, arin.CodeBadRequest, r.Method)
	}
}

func (s *Server) irrList(w http.ResponseWriter, kind string) {
	switch kind {
	case "routes":
		items := make([]arin.IRRRoute, 0, len(s.irrRoutes))
		for _, v := range s.irrRoutes {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Prefix < items[j].Prefix })
		s.writeCollection(w, func(e *xml.Encoder) error {
			for i := range items {
				if err := e.Encode(&items[i]); err != nil {
					return err
				}
			}
			return nil
		})
	case "aut-nums":
		items := make([]arin.IRRAutNum, 0, len(s.irrAutNums))
		for _, v := range s.irrAutNums {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ASNumber < items[j].ASNumber })
		s.writeCollection(w, func(e *xml.Encoder) error {
			for i := range items {
				if err := e.Encode(&items[i]); err != nil {
					return err
				}
			}
			return nil
		})
	case "as-sets":
		items := make([]arin.IRRASSet, 0, len(s.irrASSets))
		for _, v := range s.irrASSets {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		s.writeCollection(w, func(e *xml.Encoder) error {
			for i := range items {
				if err := e.Encode(&items[i]); err != nil {
					return err
				}
			}
			return nil
		})
	case "route-sets":
		items := make([]arin.IRRRouteSet, 0, len(s.irrRouteSets))
		for _, v := range s.irrRouteSets {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		s.writeCollection(w, func(e *xml.Encoder) error {
			for i := range items {
				if err := e.Encode(&items[i]); err != nil {
					return err
				}
			}
			return nil
		})
	default:
		s.fail(w, http.StatusNotFound, arin.CodeUnspecified, "list "+kind)
	}
}

// writeCollection streams a core/v1 collection wrapper around items
func (s *Server) writeCollection(w http.ResponseWriter, body func(*xml.Encoder) error) {
	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write([]byte(`<collection xmlns="http://www.arin.net/regrws/core/v1">`)); err != nil {
		panic(err)
	}
	e := xml.NewEncoder(w)
	if err := body(e); err != nil {
		panic(err)
	}
	if err := e.Flush(); err != nil {
		panic(err)
	}
	if _, err := w.Write([]byte(`</collection>`)); err != nil {
		panic(err)
	}
}
