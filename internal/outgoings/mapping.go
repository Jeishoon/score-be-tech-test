package outgoings

// Outgoings are the values recorded against a case, as monthly amounts.
type Outgoings struct {
	Water                               float64 `json:"outgoings_water"`
	Communications                      float64 `json:"outgoings_communications"`
	MortgageRent                        float64 `json:"outgoings_mortgage_rent"`
	Insurance                           float64 `json:"outgoings_insurance"`
	Investments                         float64 `json:"outgoings_investments"`
	CouncilTax                          float64 `json:"outgoings_council_tax"`
	Food                                float64 `json:"outgoings_food"`
	Clothing                            float64 `json:"outgoings_clothing"`
	OtherLivingCosts                    float64 `json:"outgoings_other_living_costs"`
	Entertainment                       float64 `json:"outgoings_entertainment"`
	Holidays                            float64 `json:"outgoings_holidays"`
	Sports                              float64 `json:"outgoings_sports"`
	Pension                             float64 `json:"outgoings_pension"`
	CarCosts                            float64 `json:"outgoings_car_costs"`
	OtherTransportCosts                 float64 `json:"outgoings_other_transport_costs"`
	ChildCare                           float64 `json:"outgoings_child_care"`
	Fuel                                float64 `json:"outgoings_fuel"`
	GroundRentServiceChargeSharedEquity float64 `json:"outgoings_ground_rent_service_charge_shared_equity_rent"`
	TelevisionLicense                   float64 `json:"outgoings_television_license"`
	HouseholdRepairs                    float64 `json:"outgoings_household_repairs"`
	AdditionalDetails                   string  `json:"outgoings_additional_details"`
}

// mapping says which ONS expenditure codes (ONS Family Spending table A35,
// expenditure by region) add up to each outgoing. A code must not appear in
// more than one field, and a parent code must not be listed together with
// its children; TestMappingNoOverlap checks both.
//
// Codes deliberately not used:
//   - 4.1.1/4.1.2 gross rent and housing benefit (4.1.3 net rent is used)
//   - 7.1 purchase of vehicles: one-off capital spending, usually recorded
//     as committed credit rather than an outgoing
//   - 14.3/14.4 income tax and national insurance: deducted from income
//   - 14.7 paying off loans, 14.8 gambling windfalls
//
// Outgoings with no matching ONS line are left at 0:
//   - ground rent / service charge / shared equity rent
type MapItem struct {
	field func(*Outgoings) *float64
	codes []string
}

var mapping = []MapItem {
	{func(o *Outgoings) *float64 { return &o.Water }, []string{"4.3"}},
	{func(o *Outgoings) *float64 { return &o.Communications }, []string{"8"}},
	// Net rent plus mortgage capital and dwelling purchase/alteration.
	{func(o *Outgoings) *float64 { return &o.MortgageRent }, []string{"4.1.3", "14.5"}},
	// Household, medical and travel insurance; vehicle insurance is in car costs.
	{func(o *Outgoings) *float64 { return &o.Insurance }, []string{"12.4.1", "12.4.2", "12.4.4", "14.2"}},
	{func(o *Outgoings) *float64 { return &o.Investments }, []string{"14.6"}},
	// ONS publishes "Housing: mortgage interest payments, council tax etc."
	// as one line; it is the closest match for council tax.
	{func(o *Outgoings) *float64 { return &o.CouncilTax }, []string{"13.1"}},
	{func(o *Outgoings) *float64 { return &o.Food }, []string{"1"}},
	{func(o *Outgoings) *float64 { return &o.Clothing }, []string{"3"}},
	// Alcohol & tobacco, household goods, health, education, personal care,
	// personal effects, other services, licences/fines, money transfers.
	{func(o *Outgoings) *float64 { return &o.OtherLivingCosts }, []string{"2", "5", "6", "10", "12.1", "12.2", "12.5", "13.2", "13.4"}},
	// Recreation & culture (minus sport, TV licences and package holidays)
	// plus restaurants and cafés.
	{func(o *Outgoings) *float64 { return &o.Entertainment }, []string{
		"9.1", "9.2", "9.3.1", "9.3.2", "9.3.4", "9.3.5",
		"9.4.2", "9.4.4", "9.4.5", "9.4.6", "9.5", "11.1",
	}},
	{func(o *Outgoings) *float64 { return &o.Holidays }, []string{"9.6", "11.2", "13.3"}},
	{func(o *Outgoings) *float64 { return &o.Sports }, []string{"9.3.3", "9.4.1"}},
	{func(o *Outgoings) *float64 { return &o.Pension }, []string{"14.1"}},
	// Running costs (spares, petrol, repairs, parking etc.) and vehicle insurance.
	{func(o *Outgoings) *float64 { return &o.CarCosts }, []string{"7.2", "12.4.3"}},
	{func(o *Outgoings) *float64 { return &o.OtherTransportCosts }, []string{"7.3"}},
	// Social protection covers childcare payments, nursery and crèche fees.
	{func(o *Outgoings) *float64 { return &o.ChildCare }, []string{"12.3"}},
	// Household fuel: electricity, gas and other fuels.
	{func(o *Outgoings) *float64 { return &o.Fuel }, []string{"4.4"}},
	// ONS combines TV licences with TV/satellite rental and subscriptions.
	{func(o *Outgoings) *float64 { return &o.TelevisionLicense }, []string{"9.4.3"}},
	{func(o *Outgoings) *float64 { return &o.HouseholdRepairs }, []string{"4.2"}},
}
