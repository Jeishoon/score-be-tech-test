package outgoings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jasonzou/score-be-tech-test/internal/cases"
	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

// ErrNoRegion means no client on the case has a postcode that resolves to an
// ONS region, so no estimate can be made.
var ErrNoRegion = errors.New("could not determine region")

// Dependencies are small interfaces so tests can swap in fakes.
type (
	CaseRepository interface {
		Get(ctx context.Context, id string) (cases.Case, error)
	}
	RegionResolver interface {
		Region(postcode string) (string, error)
	}
)

// CaseEstimate is the estimate for one case.
type CaseEstimate struct {
	CaseID   string
	Postcode string // the postcode of the first client whose region resolved
	// Sources lists each distinct region used, with the postcode it came from.
	Sources []RegionSource
	Estimate
}

// RegionSource is a region and the client postcode it was resolved from.
type RegionSource struct {
	Region   string
	Postcode string
}

// Service estimates outgoings for a case: it fetches the case, works out the
// household's region and runs the estimator.
type Service struct {
	cases     CaseRepository
	regions   RegionResolver
	estimator *Estimator
}

func NewService(c CaseRepository, r RegionResolver, e *Estimator) *Service {
	return &Service{cases: c, regions: r, estimator: e}
}

// EstimateCase returns the estimate for caseID. Errors wrap cases.ErrNotFound when
// the case does not exist and ErrNoRegion when no region can be found.
func (s *Service) EstimateCase(ctx context.Context, caseID string) (CaseEstimate, error) {
	c, err := s.cases.Get(ctx, caseID)
	if err != nil {
		return CaseEstimate{}, err
	}
	sources, err := s.householdRegions(c)
	if err != nil {
		return CaseEstimate{}, err
	}
	regions := make([]string, len(sources))
	for i, src := range sources {
		regions[i] = src.Region
	}
	est, err := s.estimator.Estimate(regions...)
	if err != nil {
		return CaseEstimate{}, err
	}
	return CaseEstimate{CaseID: c.CaseID, Postcode: sources[0].Postcode, Sources: sources, Estimate: est}, nil
}

// householdRegions returns the distinct regions of the case's clients, in
// client order, each with the first postcode that resolved to it. Clients'
// addresses are where they live now and may differ (e.g. two people moving in
// together), so the estimate averages over all of their regions. Clients with
// a missing, invalid or unknown postcode are skipped; ErrNoRegion is returned
// only if no client resolves.
func (s *Service) householdRegions(c cases.Case) ([]RegionSource, error) {
	var sources []RegionSource
	seen := map[string]bool{}
	var problems []string
	for _, cl := range c.Clients {
		raw := strings.TrimSpace(cl.Details.Address.Postcode)
		if raw == "" {
			continue
		}
		region, err := s.regions.Region(raw)
		if err != nil {
			if !errors.Is(err, postcode.ErrInvalidPostcode) && !errors.Is(err, postcode.ErrUnknownPostcode) {
				return nil, err
			}
			problems = append(problems, err.Error())
			continue
		}
		if !seen[region] {
			seen[region] = true
			sources = append(sources, RegionSource{Region: region, Postcode: raw})
		}
	}
	if len(sources) > 0 {
		return sources, nil
	}
	if len(problems) == 0 {
		return nil, fmt.Errorf("%w: no client on the case has a postcode", ErrNoRegion)
	}
	return nil, fmt.Errorf("%w: %s", ErrNoRegion, strings.Join(problems, "; "))
}
