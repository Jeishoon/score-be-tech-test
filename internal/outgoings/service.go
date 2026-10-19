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
	Postcode string // the postcode the region was taken from
	Estimate
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
	region, pc, err := s.householdRegion(c)
	if err != nil {
		return CaseEstimate{}, err
	}
	est, err := s.estimator.Estimate(region)
	if err != nil {
		return CaseEstimate{}, err
	}
	return CaseEstimate{CaseID: c.CaseID, Postcode: pc, Estimate: est}, nil
}

// householdRegion picks the region for the case. ONS figures are per
// household, so one region is used for the whole case: that of the first
// client whose postcode can be resolved.
func (s *Service) householdRegion(c cases.Case) (region, pc string, err error) {
	var problems []string
	for _, cl := range c.Clients {
		raw := strings.TrimSpace(cl.Details.Address.Postcode)
		if raw == "" {
			continue
		}
		region, err := s.regions.Region(raw)
		if err == nil {
			return region, raw, nil
		}
		if !errors.Is(err, postcode.ErrInvalidPostcode) && !errors.Is(err, postcode.ErrUnknownPostcode) {
			return "", "", err
		}
		problems = append(problems, err.Error())
	}
	if len(problems) == 0 {
		return "", "", fmt.Errorf("%w: no client on the case has a postcode", ErrNoRegion)
	}
	return "", "", fmt.Errorf("%w: %s", ErrNoRegion, strings.Join(problems, "; "))
}
