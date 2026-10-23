// Package outgoings estimates a household's monthly outgoings from ONS
// regional expenditure data.
package outgoings

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jasonzou/score-be-tech-test/internal/ons"
)

// ONS figures are weekly; outgoings are recorded monthly.
const weeksPerMonth = 52.0 / 12.0

// Estimate is the result for one region, or the average of several.
type Estimate struct {
	// Region names the region the figures are for; when several regions were
	// averaged it lists them, comma-separated.
	Region string
	// Regions lists every region the figures were averaged over.
	Regions   []string
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

// Estimate returns monthly outgoings, rounded to pennies. With several
// regions each figure is the average of their weekly values; duplicates are
// ignored. At least one region is required.
func (e *Estimator) Estimate(regions ...string) (Estimate, error) {
	regions = distinct(regions)
	if len(regions) == 0 {
		return Estimate{}, errors.New("estimate: no region given")
	}
	est := Estimate{Region: strings.Join(regions, ", "), Regions: regions}
	fellBackCodes := map[string]bool{}
	for _, m := range mapping {
		var weekly float64
		for _, region := range regions {
			for _, code := range m.codes {
				v, fellBack, err := e.ds.Value(code, region)
				if err != nil {
					return Estimate{}, err
				}
				if fellBack {
					fellBackCodes[code] = true
				}
				weekly += v
			}
		}
		*m.field(&est.Outgoings) = monthly(weekly / float64(len(regions)))
	}
	for code := range fellBackCodes {
		est.UKFallbackCodes = append(est.UKFallbackCodes, code)
	}
	sort.Strings(est.UKFallbackCodes)
	est.Outgoings.AdditionalDetails = fmt.Sprintf(
		"Estimated from ONS Family Spending %d average household expenditure %s. Monthly amounts in %s.",
		e.ds.Year, describeRegions(regions), e.ds.Currency)
	return est, nil
}

// describeRegions phrases the regions for AdditionalDetails: "for London" or
// "across London and Wales".
func describeRegions(regions []string) string {
	if len(regions) == 1 {
		return "for " + regions[0]
	}
	last := len(regions) - 1
	return "across " + strings.Join(regions[:last], ", ") + " and " + regions[last]
}

// distinct returns regions without duplicates, keeping first-seen order.
func distinct(regions []string) []string {
	seen := make(map[string]bool, len(regions))
	var out []string
	for _, r := range regions {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

func monthly(weekly float64) float64 {
	return math.Round(weekly*weeksPerMonth*100) / 100
}
