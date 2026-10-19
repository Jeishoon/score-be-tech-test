# Acre Technical Exercise

## Business Requirement Summary

When applying for a mortgage, consideration needs to be given by both lenders and intermediaries to what is known as affordability. 

Advisors and/or customers often estimate household spending as part of this exercise and record against a number of categories, however there is a significant amount of ONS data available that could be used to recommend values. 

At present the values that are recorded are: 


```
outgoings_water 

outgoings_communications 

outgoings_mortgage_rent 

outgoings_insurance 

outgoings_investments 

outgoings_council_tax 

outgoings_food 

outgoings_clothing 

outgoings_other_living_costs 

outgoings_entertainment 

outgoings_holidays 

outgoings_sports 

outgoings_pension 

outgoings_car_costs 

outgoings_other_transport_costs 

outgoings_child_care 

outgoings_fuel 

outgoings_ground_rent_service_charge_shared_equity_rent 

outgoings_television_license 

outgoings_household_repairs 

outgoings_additional_details 
```
 

 

Three ONS datasets (Expenditure by Age, Expenditure by Region and Expenditure by Composition) are present in the repo.

In order to streamline the affordability process, we would like to estimate these values based upon the provision of a Case ID. 

An example Case entity that is returned for a given ID is also provided as an attachment (`case.json`).


### Part 1 – Creating a ONS API

As an initial version of this API, we would like to create a REST API that when given a Case ID, returns the relevant values to be stored in accordance with the structure outlined above. (`outgoings_water` etc.) based upon the *region* of the clients. 

This will probably involve the following sub-tasks:

  - Importing the raw data 
  - A lookup from Postcode to Region. 
  - Implementation of the API

As well as delivering the above functionality, your submission should also show how consideration has been given to typical non functional requirements and maintainability. It should ideally also be written in Go.


### Extension: Hybrid Values

If you would like to extend your solution further, give consideration to how you could use one of the other datasets, in combination with the regional dataset to further estimate outgoings.

# score-be-tech-test
