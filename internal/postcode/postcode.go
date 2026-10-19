// Package postcode resolves UK postcodes to the ONS region names used by the
// expenditure datasets ("East Midlands", "London", "Scotland", ...).
//
// The lookup table is built from the ONS National Statistics Postcode Lookup
// (NSPL) by cmd/postcodeconvert. To keep it small it is keyed by postcode
// sector ("NG1 2") with a district ("NG1") fallback rather than holding every
// postcode.
package postcode

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrInvalidPostcode = errors.New("invalid postcode")
	ErrUnknownPostcode = errors.New("postcode not found")
)

// Outward code (area + district, e.g. "NG1", "SW1A", "M60") followed by the
// inward code (sector digit + unit letters, e.g. "2ET").
var postcodeRe = regexp.MustCompile(`^([A-Z]{1,2}[0-9][A-Z0-9]?)([0-9])([A-Z]{2})$`)

// Postcode is a validated UK postcode split into its parts.
type Postcode struct {
	Outward string // "NG1"
	Inward  string // "2ET"
}

// Parse normalises (case, spacing) and validates a postcode.
func Parse(raw string) (Postcode, error) {
	s := strings.ToUpper(strings.Join(strings.Fields(raw), ""))
	m := postcodeRe.FindStringSubmatch(s)
	if m == nil {
		return Postcode{}, fmt.Errorf("%w: %q", ErrInvalidPostcode, raw)
	}
	return Postcode{Outward: m[1], Inward: m[2] + m[3]}, nil
}

// String returns the postcode in standard form, e.g. "NG1 2ET".
func (p Postcode) String() string { return p.Outward + " " + p.Inward }

// District returns the outward code, e.g. "NG1".
func (p Postcode) District() string { return p.Outward }

// Sector returns the outward code plus sector digit, e.g. "NG1 2".
func (p Postcode) Sector() string { return p.Outward + " " + p.Inward[:1] }

// Table maps postcode sectors and districts to ONS region names.
type Table struct {
	Source    string            `json:"source"`
	Note      string            `json:"note,omitempty"`
	Sectors   map[string]string `json:"sectors"`
	Districts map[string]string `json:"districts"`
}

// Region returns the ONS region for a postcode.
func (t *Table) Region(raw string) (string, error) {
	pc, err := Parse(raw)
	if err != nil {
		return "", err
	}
	if r, ok := t.Sectors[pc.Sector()]; ok {
		return r, nil
	}
	if r, ok := t.Districts[pc.District()]; ok {
		return r, nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnknownPostcode, pc)
}
