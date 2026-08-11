package arin

import (
	"context"
	"net/http"
	"strings"
)

// registry accessors for nets, delegations, orgs, and pocs
// nets and delegations are read-modify-write only, creation and removal
// stay in arin online per the mutation safety policy

// Net fetches one registry net by handle
func (c *Client) Net(ctx context.Context, handle string) (*Net, error) {
	return irrGet[Net](c, ctx, "/rest/net/"+handle)
}

// NetUpdate rewrites an existing net
// callers send back the fetched object with only safe fields changed
func (c *Client) NetUpdate(ctx context.Context, handle string, n Net) (*Net, error) {
	return irrSend(c, ctx, http.MethodPut, "/rest/net/"+handle, nil, &n)
}

// delegationPath strips the payload's trailing dot for the url
func delegationPath(name string) string {
	return "/rest/delegation/" + strings.TrimSuffix(name, ".")
}

// Delegation fetches one reverse dns delegation by zone name
func (c *Client) Delegation(ctx context.Context, name string) (*Delegation, error) {
	return irrGet[Delegation](c, ctx, delegationPath(name))
}

// DelegationUpdate rewrites an existing delegation
func (c *Client) DelegationUpdate(ctx context.Context, name string, d Delegation) (*Delegation, error) {
	return irrSend(c, ctx, http.MethodPut, delegationPath(name), nil, &d)
}

// Org fetches one registry org by handle
func (c *Client) Org(ctx context.Context, handle string) (*Org, error) {
	return irrGet[Org](c, ctx, "/rest/org/"+handle)
}

// Poc fetches one registry point of contact by handle
func (c *Client) Poc(ctx context.Context, handle string) (*Poc, error) {
	return irrGet[Poc](c, ctx, "/rest/poc/"+handle)
}
