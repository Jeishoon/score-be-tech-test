package outgoings

import (
	"context"
	"errors"
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
	regions := fakeRegions{"NG1 2ET": "East Midlands", "SW1A 2AA": "London"}
	s := newService(t, regions,
		cases.Case{CaseID: "first-wins", Clients: []cases.Client{client("NG1 2ET"), client("SW1A 2AA")}},
		cases.Case{CaseID: "skip-bad", Clients: []cases.Client{client(""), client("bad"), client("SW1A 2AA")}},
		cases.Case{CaseID: "none-resolve", Clients: []cases.Client{client("bad"), client("ZZ1 1ZZ")}},
		cases.Case{CaseID: "no-postcode", Clients: []cases.Client{client("")}},
	)
	ctx := context.Background()

	got, err := s.EstimateCase(ctx, "first-wins")
	if err != nil || got.Region != "East Midlands" || got.Postcode != "NG1 2ET" || got.Outgoings.Food != 265.20 {
		t.Errorf("first-wins = %+v, %v", got, err)
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
