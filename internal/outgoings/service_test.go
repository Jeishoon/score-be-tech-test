package outgoings

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jasonzou/score-be-tech-test/internal/cases"
	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

// fakeRegions resolves postcodes from a fixed map; anything else is unknown.
type fakeRegions map[string]string

func (f fakeRegions) Region(pc string) (string, error) {
	if pc == "bad" {
		return "", postcode.ErrInvalidPostcode
	}
	if r, ok := f[pc]; ok {
		return r, nil
	}
	return "", postcode.ErrUnknownPostcode
}

type brokenRegions struct{}

func (brokenRegions) Region(string) (string, error) { return "", errors.New("lookup down") }

func client(pc string) cases.Client {
	var cl cases.Client
	cl.Details.Address.Postcode = pc
	return cl
}

func newService(t *testing.T, r RegionResolver, list ...cases.Case) *Service {
	t.Helper()
	return NewService(cases.NewMemoryRepository(list...), r, newEstimator(t))
}

func TestEstimateCase(t *testing.T) {
	regions := fakeRegions{"NG1 2ET": "East Midlands", "NG7 2RD": "East Midlands", "SW1A 2AA": "London"}
	s := newService(t, regions,
		cases.Case{CaseID: "mixed-regions", Clients: []cases.Client{client("NG1 2ET"), client("SW1A 2AA")}},
		cases.Case{CaseID: "same-region", Clients: []cases.Client{client("NG1 2ET"), client("NG7 2RD")}},
		cases.Case{CaseID: "skip-bad", Clients: []cases.Client{client(""), client("bad"), client("SW1A 2AA")}},
		cases.Case{CaseID: "none-resolve", Clients: []cases.Client{client("bad"), client("ZZ1 1ZZ")}},
		cases.Case{CaseID: "no-postcode", Clients: []cases.Client{client("")}},
	)
	ctx := context.Background()

	// Clients in different regions: figures are averaged across both, and
	// both regions are reported in client order.
	got, err := s.EstimateCase(ctx, "mixed-regions")
	want := []RegionSource{{"East Midlands", "NG1 2ET"}, {"London", "SW1A 2AA"}}
	if err != nil || !reflect.DeepEqual(got.Sources, want) || got.Postcode != "NG1 2ET" {
		t.Errorf("mixed-regions = %+v, %v", got, err)
	}
	avg, _ := s.estimator.Estimate("East Midlands", "London")
	if got.Outgoings != avg.Outgoings {
		t.Errorf("mixed-regions outgoings = %+v, want average %+v", got.Outgoings, avg.Outgoings)
	}

	// Different addresses in the same region: one region, unchanged figures.
	got, err = s.EstimateCase(ctx, "same-region")
	if err != nil || len(got.Sources) != 1 || got.Region != "East Midlands" || got.Postcode != "NG1 2ET" || got.Outgoings.Food != 265.20 {
		t.Errorf("same-region = %+v, %v", got, err)
	}
	got, err = s.EstimateCase(ctx, "skip-bad")
	if err != nil || got.Region != "London" {
		t.Errorf("skip-bad = %+v, %v", got, err)
	}
	for _, id := range []string{"none-resolve", "no-postcode"} {
		if _, err := s.EstimateCase(ctx, id); !errors.Is(err, ErrNoRegion) {
			t.Errorf("%s err = %v, want ErrNoRegion", id, err)
		}
	}
	if _, err := s.EstimateCase(ctx, "missing"); !errors.Is(err, cases.ErrNotFound) {
		t.Errorf("missing err = %v, want cases.ErrNotFound", err)
	}
}

func TestEstimateCaseLookupFailureIsNotNoRegion(t *testing.T) {
	s := newService(t, brokenRegions{}, cases.Case{CaseID: "a", Clients: []cases.Client{client("NG1 2ET")}})
	_, err := s.EstimateCase(context.Background(), "a")
	if err == nil || errors.Is(err, ErrNoRegion) {
		t.Errorf("err = %v, want an infrastructure error, not ErrNoRegion", err)
	}
}
