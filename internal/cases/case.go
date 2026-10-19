// Package cases provides access to mortgage Case entities.
package cases

import "errors"

var ErrNotFound = errors.New("case not found")

// Case mirrors the Case entity returned by the case service (see case.json).
type Case struct {
	CaseID  string   `json:"case_id"`
	Clients []Client `json:"clients"`
}

type Client struct {
	ClientID string  `json:"client_id"`
	Details  Details `json:"details"`
}

type Details struct {
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Address          Address `json:"address"`
	DOB              string  `json:"dob"`
	Dependents       int     `json:"dependents"`
	EmploymentStatus string  `json:"employment_status"`
}

type Address struct {
	Address1 string `json:"address1"`
	Posttown string `json:"posttown"`
	Postcode string `json:"postcode"`
}
