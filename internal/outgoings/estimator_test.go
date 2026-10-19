package outgoings

import (
	"errors"
	"strings"
	"testing"

	"github.com/jasonzou/score-be-tech-test/internal/ons"
)

func newEstimator(t *testing.T) *Estimator {
	t.Helper()
	ds, err := ons.Region2019()
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}
	e, err := NewEstimator(ds)
	if err != nil {
		t.Fatalf("NewEstimator: %v", err)
	}
	return e
}

func TestEstimateEastMidlands(t *testing.T) {
	est, err := newEstimator(t).Estimate("East Midlands")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	o := est.Outgoings
	// Weekly East Midlands figures from ONSByRegion2019.xlsx, times 52/12.
	checks := map[string]struct{ got, want float64 }{
		"food":        {o.Food, 265.20},            // 1: 61.2
		"water":       {o.Water, 37.27},            // 4.3: 8.6
		"council tax": {o.CouncilTax, 193.27},      // 13.1: 44.6
		"tv licence":  {o.TelevisionLicense, 27.3}, // 9.4.3: 6.3
		"ground rent": {o.GroundRentServiceChargeSharedEquity, 0},
	}
	for name, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", name, c.got, c.want)
		}
	}
	if !strings.Contains(o.AdditionalDetails, "East Midlands") {
		t.Errorf("additional details = %q", o.AdditionalDetails)
	}
}

func TestEstimateUnknownRegion(t *testing.T) {
	if _, err := newEstimator(t).Estimate("Atlantis"); !errors.Is(err, ons.ErrUnknownRegion) {
		t.Errorf("err = %v, want ErrUnknownRegion", err)
	}
}

func TestEstimateUsesUKFallback(t *testing.T) {
	ds, err := ons.Parse([]byte(`{
		"year": 2019, "currency": "GBP", "regions": ["UK", "London"],
		"items": [` + allMappedItems(`{"UK": 1, "London": null}`) + `]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	e, err := NewEstimator(ds)
	if err != nil {
		t.Fatalf("NewEstimator: %v", err)
	}
	est, err := e.Estimate("London")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if est.Outgoings.Food != monthly(1) {
		t.Errorf("food = %v, want UK value %v", est.Outgoings.Food, monthly(1))
	}
	if len(est.UKFallbackCodes) == 0 {
		t.Error("want fallback codes reported")
	}
}

func TestNewEstimatorRejectsMissingCodes(t *testing.T) {
	ds, err := ons.Parse([]byte(`{"regions":["UK"],"items":[{"code":"1","values":{"UK":1}}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := NewEstimator(ds); err == nil {
		t.Error("want error for dataset missing mapped codes")
	}
}

// Every ONS code may be used by at most one outgoing, and a code may not be
// listed alongside its own parent or child, or spending would be counted twice.
func TestMappingNoOverlap(t *testing.T) {
	var all []string
	for _, m := range mapping {
		all = append(all, m.codes...)
	}
	for i, a := range all {
		for _, b := range all[i+1:] {
			if a == b || strings.HasPrefix(a, b+".") || strings.HasPrefix(b, a+".") {
				t.Errorf("codes %q and %q overlap", a, b)
			}
		}
	}
}

func allMappedItems(values string) string {
	var items []string
	for _, m := range mapping {
		for _, c := range m.codes {
			items = append(items, `{"code":"`+c+`","values":`+values+`}`)
		}
	}
	return strings.Join(items, ",")
}
