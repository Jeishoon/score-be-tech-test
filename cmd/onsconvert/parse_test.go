package main

import (
	"testing"
)

// Rows shaped like the real sheet, two regions only.
var sampleRows = [][]string{
	{"", "", "", "", "UK", "London"},
	{"1", "Food & non-alcoholic drinks", "", "", "60.3", "65.8"},
	{},
	{"", "1.1.1", "Bread, rice and cereals", "", "5.4000000000000004", "5.9"},
	{"", "2.1.4", "Alcopops (brought home)", "", "0", "[0.00~]"},
	{"", "3.1.11", "Dry cleaners", "", "0.2", "[0.20]"},
	{"", "", "", "", ":", ":"}, // stray marker row
	{"4.0999999999999996", "Actual rentals for housing", "", "", "49.2", "60.1"},
	{"", "4.1.3", "Net rent2", "", "35.4", "50.0"},
	{"", "4.1.4", "Second dwelling rent", "", "[0.10]", ".."},
	{"4.3", "Water supply and miscellaneous services", "", "", "", ""},
	{"", "relating to the dwelling", "", "", "9.3", "10.1"},
	{"1-12", "All expenditure groups", "", "", "496.6", "600"},
	{"Total expenditure ", "", "", "", "571.79999999999995", "686"},
	{"14", "Other items recorded"},
	{"14.1", "Life assurance", "", "", "25.7", "30"},
}

func TestParse(t *testing.T) {
	d, err := Parse(sampleRows, Options{Source: "test.xlsx", Year: 2019, Currency: "GBP"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got, want := d.Regions, []string{"UK", "London"}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("regions = %v, want %v", got, want)
	}
	if got := d.TotalExpenditure["UK"]; got != 571.8 {
		t.Errorf("UK total = %v, want 571.8", got)
	}

	wantCodes := []string{"1", "1.1.1", "2.1.4", "3.1.11", "4.1", "4.1.3", "4.1.4", "4.3", "14.1"}
	if len(d.Items) != len(wantCodes) {
		var got []string
		for _, it := range d.Items {
			got = append(got, it.Code)
		}
		t.Fatalf("codes = %v, want %v", got, wantCodes)
	}
	for i, c := range wantCodes {
		if d.Items[i].Code != c {
			t.Errorf("item %d code = %q, want %q", i, d.Items[i].Code, c)
		}
	}

	byCode := map[string]int{}
	for i, it := range d.Items {
		byCode[it.Code] = i
	}
	val := func(code, region string) *float64 { return d.Items[byCode[code]].Values[region] }

	if v := val("1.1.1", "UK"); v == nil || *v != 5.4 {
		t.Errorf("1.1.1 UK = %v, want 5.4", v)
	}
	if v := val("2.1.4", "London"); v == nil || *v != 0 {
		t.Errorf("2.1.4 London = %v, want 0", v)
	}
	if v := val("3.1.11", "London"); v == nil || *v != 0.2 {
		t.Errorf("3.1.11 London = %v, want 0.2", v)
	}
	if v := val("4.1.4", "London"); v != nil {
		t.Errorf("4.1.4 London = %v, want nil", *v)
	}
	if got := d.Items[byCode["4.1.3"]].Label; got != "Net rent" {
		t.Errorf("4.1.3 label = %q, want footnote stripped", got)
	}
	water := d.Items[byCode["4.3"]]
	if water.Label != "Water supply and miscellaneous services relating to the dwelling" {
		t.Errorf("4.3 label = %q, want wrapped label joined", water.Label)
	}
	if v := water.Values["London"]; v == nil || *v != 10.1 {
		t.Errorf("4.3 London = %v, want 10.1 from continuation row", v)
	}
}

func TestParseValue(t *testing.T) {
	cases := []struct {
		in   string
		want *float64
		err  bool
	}{
		{"5.4", ptr(5.4), false},
		{"4.9000000000000004", ptr(4.9), false},
		{"[0.20]", ptr(0.2), false},
		{"[0.00~]", ptr(0), false},
		{"..", nil, false},
		{":", nil, false},
		{"n/a", nil, true},
	}
	for _, c := range cases {
		got, err := ParseValue(c.in)
		if (err != nil) != c.err {
			t.Errorf("ParseValue(%q) err = %v, want err %v", c.in, err, c.err)
			continue
		}
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("ParseValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	if _, err := Parse(nil, Options{}); err == nil {
		t.Error("want error for empty sheet")
	}
	rows := [][]string{{"", "", "", "", "UK"}, {"1", "Food", "", "", "abc"}}
	if _, err := Parse(rows, Options{}); err == nil {
		t.Error("want error for bad value")
	}
}

func ptr(f float64) *float64 { return &f }
