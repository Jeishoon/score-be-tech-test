// Package ons holds the in-memory model of an ONS "Family Spending" expenditure
// table, as produced by cmd/onsconvert.
package ons

import (
	"encoding/json"
	"errors"
	"fmt"
)

// UK is the column used as a fallback when a region has no figure for a code.
const UK = "UK"

// Dataset is one ONS expenditure table (e.g. expenditure by region).
// Values are average weekly household expenditure in Currency.
type Dataset struct {
	Source           string             `json:"source"`
	Year             int                `json:"year"`
	Currency         string             `json:"currency"`
	Regions          []string           `json:"regions"`
	TotalExpenditure map[string]float64 `json:"total_expenditure"`
	Items            []Item             `json:"items"`

	byCode map[string]Item
}

// Item is one expenditure category, e.g. code "4.3" "Water supply ...".
// A nil value means ONS did not publish a figure for that region.
type Item struct {
	Code   string              `json:"code"`
	Label  string              `json:"label"`
	Values map[string]*float64 `json:"values"`
}

var (
	ErrUnknownCode   = errors.New("unknown ONS code")
	ErrUnknownRegion = errors.New("unknown region")
	ErrNoValue       = errors.New("no value published")
)

// Parse decodes and validates a dataset produced by cmd/onsconvert.
func Parse(data []byte) (*Dataset, error) {
	var d Dataset
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("decode dataset: %w", err)
	}
	if err := d.index(); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Dataset) index() error {
	if len(d.Items) == 0 {
		return errors.New("dataset has no items")
	}
	if !d.HasRegion(UK) {
		return errors.New("dataset has no UK column")
	}
	d.byCode = make(map[string]Item, len(d.Items))
	for _, it := range d.Items {
		if _, dup := d.byCode[it.Code]; dup {
			return fmt.Errorf("duplicate code %q", it.Code)
		}
		d.byCode[it.Code] = it
	}
	return nil
}

// HasRegion reports whether the dataset has a column for region.
func (d *Dataset) HasRegion(region string) bool {
	for _, r := range d.Regions {
		if r == region {
			return true
		}
	}
	return false
}

// Item returns the item for code.
func (d *Dataset) Item(code string) (Item, bool) {
	it, ok := d.byCode[code]
	return it, ok
}

// Value returns the weekly figure for code in region. If ONS did not publish a
// figure for that region (sample too small), the UK figure is used instead and
// fellBack is true.
func (d *Dataset) Value(code, region string) (v float64, fellBack bool, err error) {
	if !d.HasRegion(region) {
		return 0, false, fmt.Errorf("%w: %q", ErrUnknownRegion, region)
	}
	it, ok := d.byCode[code]
	if !ok {
		return 0, false, fmt.Errorf("%w: %q", ErrUnknownCode, code)
	}
	if p := it.Values[region]; p != nil {
		return *p, false, nil
	}
	if p := it.Values[UK]; p != nil {
		return *p, true, nil
	}
	return 0, false, fmt.Errorf("%w: code %q", ErrNoValue, code)
}
