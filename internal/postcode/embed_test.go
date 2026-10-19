package postcode

import "testing"

func TestLoadTable(t *testing.T) {
	tbl, err := LoadTable()
	if err != nil {
		t.Fatalf("LoadTable: %v", err)
	}
	cases := map[string]string{
		"NG1 2ET":  "East Midlands",
		"SW1A 1AA": "London",
		"EH1 1YZ":  "Scotland",
		"CF10 1EP": "Wales",
		"BT1 5GS":  "Northern Ireland",
		"M1 1AE":   "North West",
		"LS1 4AP":  "Yorkshire and the Humber",
	}
	for in, want := range cases {
		if got, err := tbl.Region(in); err != nil || got != want {
			t.Errorf("Region(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
