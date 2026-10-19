// Package outgoings estimates a household's monthly outgoings from ONS
// regional expenditure data.
package outgoings

import (
	"fmt"
	"math"
	"sort"

	"github.com/jasonzou/score-be-tech-test/internal/ons"
)

// ONS figures are weekly; outgoings are recorded monthly.
const weeksPerMonth = 52.0 / 12.0

// Estimate is the result for one region.
type Estimate struct {
	Region    string
	Outgoings Outgoings
	// UKFallbackCodes lists ONS codes that had no figure for the region, so
	// the UK average was used instead.
	UKFallbackCodes []string
}

// Estimator turns a region into estimated monthly outgoings.
type Estimator struct {
	ds *ons.Dataset
}

// NewEstimator checks that every mapped code exists in the dataset, so a
// mapping or data mistake fails at startup rather than on a request.
func NewEstimator(ds *ons.Dataset) (*Estimator, error) {
	for _, m := range mapping {
		for _, code := range m.codes {
			if _, ok := ds.Item(code); !ok {
				return nil, fmt.Errorf("mapping uses ONS code %q not in dataset %s", code, ds.Source)
			}
		}
	}
	return &Estimator{ds: ds}, nil
}

// Estimate returns monthly outgoings for region, rounded to pennies.
func (e *Estimator) Estimate(region string) (Estimate, error) {
	est := Estimate{Region: region}
	for _, m := range mapping {
		var weekly float64
		for _, code := range m.codes {
			v, fellBack, err := e.ds.Value(code, region)
			if err != nil {
				return Estimate{}, err
			}
			if fellBack {
				est.UKFallbackCodes = append(est.UKFallbackCodes, code)
			}
			weekly += v
		}
		*m.field(&est.Outgoings) = monthly(weekly)
	}
	sort.Strings(est.UKFallbackCodes)
	est.Outgoings.AdditionalDetails = fmt.Sprintf(
		"Estimated from ONS Family Spending %d average household expenditure for %s. Monthly amounts in %s.",
		e.ds.Year, region, e.ds.Currency)
	return est, nil
}

func monthly(weekly float64) float64 {
	return math.Round(weekly*weeksPerMonth*100) / 100
}
