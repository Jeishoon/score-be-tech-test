package main

import (
	"errors"
	"testing"

	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

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
	if _, err := tbl.Region("ZZ9 9ZZ"); !errors.Is(err, postcode.ErrUnknownPostcode) {
		t.Errorf("unknown district err = %v", err)
	}
	if _, err := tbl.Region("nope"); !errors.Is(err, postcode.ErrInvalidPostcode) {
		t.Errorf("invalid postcode err = %v", err)
	}
}
