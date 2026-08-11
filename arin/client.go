// Package arin is a client for the arin rpki restful api
package arin

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultBaseURL = "https://reg.arin.net"
	OTEBaseURL     = "https://reg.ote.arin.net"

	// responses are small, cap reads defensively
	maxBody = 10 << 20
)

// Client talks to the arin rpki restful api for a single org
type Client struct {
	base *url.URL
	key  string
	org  string
	http *http.Client
}

// New validates inputs and returns a client bound to base, key, and org
func New(base, key, org string) (*Client, error) {
	if key == "" {
		return nil, fmt.Errorf("arin: empty api key")
	}
	if org == "" {
		return nil, fmt.Errorf("arin: empty org handle")
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("arin: base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("arin: base url %q missing scheme or host", base)
	}
	return &Client{base: u, key: key, org: org, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Org returns the org handle the client is bound to
func (c *Client) Org() string { return c.org }

// do sends one request and returns the raw response body
// non-2xx responses decode into *Error when the body is a reg-rws
// error payload
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	var rd io.Reader
	if body != nil {
		b, err := xml.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("arin: marshal: %w", err)
		}
		rd = bytes.NewReader(append([]byte(xml.Header), b...))
	}
	// ote only honors the legacy apikey query parameter while the
	// production docs prefer the authorization header, so send both
	u := c.base.JoinPath(path)
	q := u.Query()
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("apikey", c.key)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rd)
	if err != nil {
		return nil, fmt.Errorf("arin: request: %w", err)
	}
	req.Header.Set("Authorization", "ApiKey "+c.key)
	req.Header.Set("Accept", "application/xml")
	if body != nil {
		req.Header.Set("Content-Type", "application/xml")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arin: %s %s: %w", method, path, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("arin: read response: %w", err)
	}
	if res.StatusCode >= 300 {
		var e Error
		if xml.Unmarshal(data, &e) == nil && e.Code != "" {
			e.Status = res.StatusCode
			return nil, &e
		}
		return nil, fmt.Errorf("arin: %s %s: http %d: %.200s", method, path, res.StatusCode, data)
	}
	return data, nil
}

// Transact posts an all-or-nothing rpki transaction
func (c *Client) Transact(ctx context.Context, tx Transaction) error {
	_, err := c.do(ctx, http.MethodPost, "/rest/rpki/"+c.org, nil, &tx)
	return err
}

// ROAs lists every roa spec for the org
func (c *Client) ROAs(ctx context.Context) ([]ROASpec, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/roa/"+c.org, nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[ROASpec](data, "roaSpec")
}

// ASPAs lists every aspa for the org
func (c *Client) ASPAs(ctx context.Context) ([]ASPA, error) {
	data, err := c.do(ctx, http.MethodGet, "/rest/aspa/"+c.org, nil, nil)
	if err != nil {
		return nil, err
	}
	return collect[ASPA](data, "aspa")
}

// collect decodes every element named name anywhere in the document
// arin leaves the list envelope unspecified, so decoding scans for
// items instead of assuming a wrapper element
func collect[T any](data []byte, name string) ([]T, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var out []T
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("arin: decode %s list: %w", name, err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != name {
			continue
		}
		var v T
		if err := d.DecodeElement(&v, &se); err != nil {
			return nil, fmt.Errorf("arin: decode %s: %w", name, err)
		}
		out = append(out, v)
	}
}
