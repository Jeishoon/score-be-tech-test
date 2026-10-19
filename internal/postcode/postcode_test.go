package postcode

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]struct{ sector, district string }{
		"NG1 2ET":   {"NG1 2", "NG1"},
		"ng12et":    {"NG1 2", "NG1"},
		" NG1  2ET": {"NG1 2", "NG1"},
		"SW1A 1AA":  {"SW1A 1", "SW1A"},
		"M60 1NW":   {"M60 1", "M60"},
		"EH1 1YZ":   {"EH1 1", "EH1"},
	}
	for in, want := range cases {
		pc, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if pc.Sector() != want.sector || pc.District() != want.district {
			t.Errorf("Parse(%q) = sector %q district %q, want %q %q", in, pc.Sector(), pc.District(), want.sector, want.district)
		}
	}
	for _, in := range []string{"", "NOTAPOSTCODE", "12345", "NG1", "NG1 2E"} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalidPostcode) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalidPostcode", in, err)
		}
	}
}

func TestBuilderAndRegion(t *testing.T) {
	b := NewBuilder()
	// NG1 2 is mostly East Midlands.
	b.Add("NG1 2ET", "E12000004", false)
	b.Add("NG1 2EU", "E12000004", false)
	b.Add("NG1 2EX", "E12000005", false)
	// KT1 1 only has a terminated postcode.
	b.Add("KT1 1AA", "E12000007", true)
	// Live postcodes win over terminated ones in the same sector.
	b.Add("CR0 1AA", "E12000008", true)
	b.Add("CR0 1AB", "E12000008", true)
	b.Add("CR0 1AD", "E12000007", false)
	// Ignored: Isle of Man, junk postcode.
	b.Add("IM1 1AA", "M99999999", false)
	b.Add("garbage", "E12000004", false)

	tbl, st := b.Table("test")
	if b.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", b.Skipped)
	}
	if st.MixedSectors != 1 {
		t.Errorf("MixedSectors = %d, want 1", st.MixedSectors)
	}

	cases := map[string]string{
		"NG1 2ET": "East Midlands",
		"NG1 2ZZ": "East Midlands", // unseen unit, same sector
		"NG1 9ZZ": "East Midlands", // unseen sector, district fallback
		"KT1 1ZZ": "London",
		"CR0 1ZZ": "London",
	}
	for in, want := range cases {
		got, err := tbl.Region(in)
		if err != nil || got != want {
			t.Errorf("Region(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := tbl.Region("ZZ9 9ZZ"); !errors.Is(err, ErrUnknownPostcode) {
		t.Errorf("unknown district err = %v", err)
	}
	if _, err := tbl.Region("nope"); !errors.Is(err, ErrInvalidPostcode) {
		t.Errorf("invalid postcode err = %v", err)
	}
}
