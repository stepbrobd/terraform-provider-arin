package arin

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// the irr restful api is documented with route6-specific shapes, but
// ote serves both families under the route root with a version field
// verified live 2026-08-11

func irrRoutePath(prefix string, origin int64) (string, error) {
	ip, length, ok := strings.Cut(prefix, "/")
	if !ok {
		return "", fmt.Errorf("arin: prefix %q missing length", prefix)
	}
	return fmt.Sprintf("/rest/irr/route/%s/%s/AS%d", ip, length, origin), nil
}

func irrGet[T any](c *Client, ctx context.Context, path string) (*T, error) {
	data, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var out T
	if err := xml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("arin: decode %s: %w", path, err)
	}
	return &out, nil
}

func irrSend[T any](c *Client, ctx context.Context, method, path string, query url.Values, body *T) (*T, error) {
	data, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	var out T
	if err := xml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("arin: decode %s: %w", path, err)
	}
	return &out, nil
}

// Route fetches one irr route object
func (c *Client) Route(ctx context.Context, prefix string, origin int64) (*IRRRoute, error) {
	p, err := irrRoutePath(prefix, origin)
	if err != nil {
		return nil, err
	}
	return irrGet[IRRRoute](c, ctx, p)
}

// RouteCreate registers a new irr route object
func (c *Client) RouteCreate(ctx context.Context, prefix string, origin int64, r IRRRoute) (*IRRRoute, error) {
	p, err := irrRoutePath(prefix, origin)
	if err != nil {
		return nil, err
	}
	return irrSend(c, ctx, http.MethodPost, p, nil, &r)
}

// RouteUpdate rewrites an existing irr route object
func (c *Client) RouteUpdate(ctx context.Context, prefix string, origin int64, r IRRRoute) (*IRRRoute, error) {
	p, err := irrRoutePath(prefix, origin)
	if err != nil {
		return nil, err
	}
	return irrSend(c, ctx, http.MethodPut, p, nil, &r)
}

// RouteDelete removes an irr route object
func (c *Client) RouteDelete(ctx context.Context, prefix string, origin int64) error {
	p, err := irrRoutePath(prefix, origin)
	if err != nil {
		return err
	}
	_, err = c.do(ctx, http.MethodDelete, p, nil, nil)
	return err
}

// Routes lists the org's irr route objects
func (c *Client) Routes(ctx context.Context) ([]IRRRoute, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/irr/org/"+c.org+"/routes", nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[IRRRoute](data, "route")
}

// AutNum fetches one irr aut-num object
func (c *Client) AutNum(ctx context.Context, as int64) (*IRRAutNum, error) {
	return irrGet[IRRAutNum](c, ctx, fmt.Sprintf("/rest/irr/aut-num/AS%d", as))
}

// AutNumCreate registers a new irr aut-num object
func (c *Client) AutNumCreate(ctx context.Context, as int64, a IRRAutNum) (*IRRAutNum, error) {
	return irrSend(c, ctx, http.MethodPost, fmt.Sprintf("/rest/irr/aut-num/AS%d", as), nil, &a)
}

// AutNumUpdate rewrites an existing irr aut-num object
func (c *Client) AutNumUpdate(ctx context.Context, as int64, a IRRAutNum) (*IRRAutNum, error) {
	return irrSend(c, ctx, http.MethodPut, fmt.Sprintf("/rest/irr/aut-num/AS%d", as), nil, &a)
}

// AutNumDelete removes an irr aut-num object
func (c *Client) AutNumDelete(ctx context.Context, as int64) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/rest/irr/aut-num/AS%d", as), nil, nil)
	return err
}

// AutNums lists the org's irr aut-num objects
func (c *Client) AutNums(ctx context.Context) ([]IRRAutNum, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/irr/org/"+c.org+"/aut-nums", nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[IRRAutNum](data, "autnum")
}

// ASSet fetches one irr as-set object
func (c *Client) ASSet(ctx context.Context, name string) (*IRRASSet, error) {
	return irrGet[IRRASSet](c, ctx, "/rest/irr/as-set/"+name)
}

// ASSetCreate registers a new irr as-set under the client org
func (c *Client) ASSetCreate(ctx context.Context, s IRRASSet) (*IRRASSet, error) {
	return irrSend(c, ctx, http.MethodPost, "/rest/irr/as-set", url.Values{"orgHandle": {c.org}}, &s)
}

// ASSetUpdate rewrites an existing irr as-set object
func (c *Client) ASSetUpdate(ctx context.Context, name string, s IRRASSet) (*IRRASSet, error) {
	return irrSend(c, ctx, http.MethodPut, "/rest/irr/as-set/"+name, nil, &s)
}

// ASSetDelete removes an irr as-set object
func (c *Client) ASSetDelete(ctx context.Context, name string) error {
	_, err := c.do(ctx, http.MethodDelete, "/rest/irr/as-set/"+name, nil, nil)
	return err
}

// ASSets lists the org's irr as-set objects
func (c *Client) ASSets(ctx context.Context) ([]IRRASSet, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/irr/org/"+c.org+"/as-sets", nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[IRRASSet](data, "asSet")
}

// RouteSet fetches one irr route-set object
func (c *Client) RouteSet(ctx context.Context, name string) (*IRRRouteSet, error) {
	return irrGet[IRRRouteSet](c, ctx, "/rest/irr/route-set/"+name)
}

// RouteSetCreate registers a new irr route-set under the client org
func (c *Client) RouteSetCreate(ctx context.Context, s IRRRouteSet) (*IRRRouteSet, error) {
	return irrSend(c, ctx, http.MethodPost, "/rest/irr/route-set", url.Values{"orgHandle": {c.org}}, &s)
}

// RouteSetUpdate rewrites an existing irr route-set object
func (c *Client) RouteSetUpdate(ctx context.Context, name string, s IRRRouteSet) (*IRRRouteSet, error) {
	return irrSend(c, ctx, http.MethodPut, "/rest/irr/route-set/"+name, nil, &s)
}

// RouteSetDelete removes an irr route-set object
func (c *Client) RouteSetDelete(ctx context.Context, name string) error {
	_, err := c.do(ctx, http.MethodDelete, "/rest/irr/route-set/"+name, nil, nil)
	return err
}

// RouteSets lists the org's irr route-set objects
func (c *Client) RouteSets(ctx context.Context) ([]IRRRouteSet, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/irr/org/"+c.org+"/route-sets", nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[IRRRouteSet](data, "routeSet")
}
