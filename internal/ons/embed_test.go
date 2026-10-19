package ons

import "testing"

func TestRegion2019Loads(t *testing.T) {
	d, err := Region2019()
	if err != nil {
		t.Fatalf("Region2019: %v", err)
	}
	if got := d.TotalExpenditure["East Midlands"]; got != 553.3 {
		t.Errorf("East Midlands total = %v, want 553.3", got)
	}
	if v, _, err := d.Value("1", "East Midlands"); err != nil || v != 61.2 {
		t.Errorf("food East Midlands = %v, %v; want 61.2", v, err)
	}
}
