package main

// This file turns the rows of an ONS "Family Spending" xlsx sheet into an
// ons.Dataset. It works on plain [][]string rows so it can be tested without
// an xlsx file; main.go does the file reading.

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/jasonzou/score-be-tech-test/internal/ons"
)

// Layout of the sheet. The code sits in column A, B or C depending on depth,
// with the label in the column after it. Region figures start at column E.
const (
	maxCodeCol  = 2 // column C
	firstValCol = 4 // column E
)

var (
	// "1", "1.1", "12.4.3"
	codeRe = regexp.MustCompile(`^\d+(\.\d+)*$`)
	// Some two-level codes are stored as numbers, so 4.1 reads back as
	// "4.0999999999999996".
	floatCodeRe = regexp.MustCompile(`^\d+\.\d{6,}$`)
	// Footnote markers glued to the end of a label: "Net rent2", "licences5".
	// Only strip when glued to a letter, so "(5-15)" style text is kept.
	footnoteRe = regexp.MustCompile(`([a-zA-Z'])\d+$`)
	spaceRe    = regexp.MustCompile(`\s+`)
)

// Options describes the dataset metadata that is not in the sheet itself.
type Options struct {
	Source   string
	Year     int
	Currency string
}

// Parse converts sheet rows into a dataset.
func Parse(rows [][]string, opts Options) (*ons.Dataset, error) {
	if len(rows) == 0 {
		return nil, errors.New("sheet is empty")
	}
	cols := headerColumns(rows[0])
	if len(cols) == 0 {
		return nil, errors.New("no region headers found in first row")
	}
	regions := make([]string, len(cols))
	for i, col := range cols {
		regions[i] = col.name
	}

	d := &ons.Dataset{
		Source:           opts.Source,
		Year:             opts.Year,
		Currency:         opts.Currency,
		Regions:          regions,
		TotalExpenditure: map[string]float64{},
	}

	// cur is the item being built. Long labels wrap onto following rows, and
	// ONS puts the figures on the last of those rows, so an item stays open
	// until a row with figures has been seen.
	var cur *ons.Item
	var curIsTotal bool
	finish := func() error {
		if cur == nil {
			return nil
		}
		defer func() { cur = nil; curIsTotal = false }()
		if cur.Values == nil {
			// Heading without figures, e.g. "14 Other items recorded".
			return nil
		}
		if curIsTotal {
			for r, v := range cur.Values {
				if v == nil {
					return fmt.Errorf("total expenditure missing for %s", r)
				}
				d.TotalExpenditure[r] = *v
			}
			return nil
		}
		cur.Label = cleanLabel(cur.Label)
		d.Items = append(d.Items, *cur)
		return nil
	}

	for i, row := range rows[1:] {
		rowNum := i + 2 // 1-based, after header
		code, label, codeCol := rowKey(row)
		vals, hasVals, err := rowValues(row, cols)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}

		switch {
		case code != "":
			if err := finish(); err != nil {
				return nil, fmt.Errorf("row %d: %w", rowNum, err)
			}
			cur = &ons.Item{Code: code, Label: label}
		case isTotalRow(row):
			if err := finish(); err != nil {
				return nil, fmt.Errorf("row %d: %w", rowNum, err)
			}
			cur, curIsTotal = &ons.Item{Code: "total"}, true
		case codeCol == -1 && label != "" && cur != nil && cur.Values == nil:
			// Continuation of a wrapped label.
			cur.Label += " " + label
		case codeCol == -1 && label != "":
			// Other text, e.g. "1-12 All expenditure groups" subtotal or notes.
			if err := finish(); err != nil {
				return nil, fmt.Errorf("row %d: %w", rowNum, err)
			}
			continue
		}

		if hasVals && cur != nil {
			if cur.Values != nil && codeCol == -1 && label == "" && allNil(vals) {
				// Stray row of ":" markers under a finished item (row 84 of
				// the 2019 sheet). It carries no information.
				continue
			}
			if cur.Values != nil {
				return nil, fmt.Errorf("row %d: second row of figures for code %q", rowNum, cur.Code)
			}
			cur.Values = vals
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	if len(d.Items) == 0 {
		return nil, errors.New("no items found")
	}
	if len(d.TotalExpenditure) == 0 {
		return nil, errors.New("no total expenditure row found")
	}
	return d, nil
}

// column is one region column of the sheet: its index and header name.
type column struct {
	idx  int
	name string
}

// headerColumns returns the region columns from the header row, in sheet order.
func headerColumns(header []string) []column {
	var cols []column
	for c := firstValCol; c < len(header); c++ {
		if name := strings.TrimSpace(header[c]); name != "" {
			cols = append(cols, column{idx: c, name: name})
		}
	}
	return cols
}

// rowKey finds the category code and label in columns A-D. If the row has no
// code, codeCol is -1 and label is any text found (a wrapped label line).
func rowKey(row []string) (code, label string, codeCol int) {
	for c := 0; c <= maxCodeCol && c < len(row); c++ {
		v := strings.TrimSpace(row[c])
		if v == "" {
			continue
		}
		if floatCodeRe.MatchString(v) {
			f, _ := strconv.ParseFloat(v, 64)
			v = strconv.FormatFloat(roundTo(f, 3), 'f', -1, 64)
		}
		if codeRe.MatchString(v) {
			return v, strings.TrimSpace(cell(row, c+1)), c
		}
		return "", v, -1
	}
	return "", strings.TrimSpace(cell(row, maxCodeCol+1)), -1
}

func isTotalRow(row []string) bool {
	return strings.EqualFold(strings.TrimSpace(cell(row, 0)), "Total expenditure")
}

// rowValues reads the region figures. hasVals is false when every region
// cell is blank (a label-only row).
func rowValues(row []string, cols []column) (map[string]*float64, bool, error) {
	vals := make(map[string]*float64, len(cols))
	hasVals := false
	for _, col := range cols {
		raw := strings.TrimSpace(cell(row, col.idx))
		if raw == "" {
			vals[col.name] = nil
			continue
		}
		hasVals = true
		v, err := ParseValue(raw)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", col.name, err)
		}
		vals[col.name] = v
	}
	return vals, hasVals, nil
}

// ParseValue converts one ONS cell to a number. ONS markers:
//
//	"[0.20]"  figure based on a small sample: keep the number
//	"[0.00~]" rounds to zero
//	".." / ":" not available: nil
func ParseValue(raw string) (*float64, error) {
	s := strings.TrimSpace(raw)
	if s == ".." || s == ":" {
		return nil, nil
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSuffix(s, "~")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("unrecognised value %q", raw)
	}
	// xlsx stores 4.9 as 4.9000000000000004; ONS publishes at most 2dp.
	v = roundTo(v, 2)
	return &v, nil
}

func roundTo(v float64, places int) float64 {
	p := math.Pow10(places)
	return math.Round(v*p) / p
}

func allNil(vals map[string]*float64) bool {
	for _, v := range vals {
		if v != nil {
			return false
		}
	}
	return true
}

func cleanLabel(s string) string {
	s = spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	return footnoteRe.ReplaceAllString(s, "$1")
}

func cell(row []string, c int) string {
	if c < len(row) {
		return row[c]
	}
	return ""
}
