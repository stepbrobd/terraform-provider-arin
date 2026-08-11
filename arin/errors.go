package arin

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// error codes arin documents for reg-rws
const (
	CodeAuthentication   = "E_AUTHENTICATION"
	CodeBadRequest       = "E_BAD_REQUEST"
	CodeEntityValidation = "E_ENTITY_VALIDATION"
	CodeSchemaValidation = "E_SCHEMA_VALIDATION"
	CodeNotRemoveable    = "E_NOT_REMOVEABLE"
	CodeObjectNotFound   = "E_OBJECT_NOT_FOUND"
	CodeOutage           = "E_OUTAGE"
	CodeTooManyRequests  = "E_TOO_MANY_REQUESTS"
	CodeUnspecified      = "E_UNSPECIFIED"
)

// Error is the reg-rws error payload
type Error struct {
	XMLName        xml.Name    `xml:"error"`
	Status         int         `xml:"-"`
	Code           string      `xml:"code"`
	Message        string      `xml:"message"`
	Components     []Component `xml:"components>component"`
	AdditionalInfo []string    `xml:"additionalInfo>message"`
}

type Component struct {
	Name    string `xml:"name"`
	Message string `xml:"message"`
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "arin: %s (%s, http %d)", e.Message, e.Code, e.Status)
	for _, c := range e.Components {
		fmt.Fprintf(&b, "; %s: %s", c.Name, c.Message)
	}
	for _, m := range e.AdditionalInfo {
		fmt.Fprintf(&b, "; %s", m)
	}
	return b.String()
}

// IsNotFound reports whether err is an arin E_OBJECT_NOT_FOUND error
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == CodeObjectNotFound
}

// IsMissing reports whether err denotes an absent object on endpoints
// where ote signals missing data with a bare 404 and E_UNSPECIFIED,
// which the irr api does
// rpki call sites keep the stricter IsNotFound
func IsMissing(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == CodeObjectNotFound || e.Status == http.StatusNotFound
}
