## Part 1

A Go REST API that estimates monthly household outgoings for a case from the ONS 2019 expenditure-by-region data.

### Running

Requires Go 1.27+.

```sh
make run # or: go run ./cmd/server
curl localhost:8080/cases/56763bf7-d046-4a54-a4a1-0728c724a092/outgoings | python3 -m json.tool
make test # or: go test ./...
```

| Env var      | Default     | Meaning                                         |
| ------------ | ----------- | ----------------------------------------------- |
| `ADDR`       | `:8080`     | listen address                                  |
| `CASES_FILE` | `case.json` | JSON file holding one case or an array of cases |

### API

`GET /cases/{caseID}/outgoings`

```json
{
"case_id": "56763bf7-d046-4a54-a4a1-0728c724a092",
"region": "East Midlands",
"postcode": "NG1 2ET",
"period": "monthly",
"currency": "GBP",
"source": "ONS Family Spending 2019, expenditure by region (ONSByRegion2019.xlsx)",
"outgoings": { "outgoings_water": 37.27, "outgoings_food": 265.2, "...": "..." }
}
```

| Status | When                                                         |
| ------ | ------------------------------------------------------------ |
| 200    | estimate returned                                            |
| 400    | case ID is not a UUID                                        |
| 404    | case not found                                               |
| 422    | no client postcode on the case could be resolved to a region |
| 500    | unexpected error (details are logged, not returned)          |
  
`uk_fallback_codes` is included when ONS published no figure for the region for some category (sample too small), so the UK average was used instead.

`GET /healthz` returns `{"status":"ok"}`.

### Data pipeline

The raw data is converted to JSON **once** and the JSON is embedded in the binary (`go:embed`), so the server does no xlsx or CSV parsing and needs no data files at runtime.

| Data | Converter | Output |
| ---- | --------- | ------ |
| `ONSByRegion2019.xlsx` | `make convert-ons` (`cmd/onsconvert`) | `internal/ons/data/region2019.json` |
| ONS National Statistics Postcode Lookup (NSPL) | `make convert-postcodes NSPL=path/to/nspl.zip` (`cmd/postcodeconvert`) | `internal/postcode/data/sectors.json` |

The xlsx converter handles the sheet's quirks: codes in column A/B/C depending on depth, labels wrapped onto a second row (where the figures sit), codes stored as floats (`4.0999999999999996`), footnote digits on labels, and the ONS markers `[x]` (small sample, value kept), `[0.00~]` (0), `..` / `:` (not available → `null`).

**Postcode → region** uses the official ONS NSPL (Open Government Licence), downloaded from the [ONS Open Geography Portal](https://geoportal.statistics.gov.uk) (search "National Statistics Postcode Lookup"; the committed table was built from the August 2026 release). The full file has 2.7M postcodes (~1GB CSV), so the converter reduces it to a postcode **sector** (`NG1 2`) → region table, with a **district** (`NG1`) fallback, by majority vote over live postcodes (~12k sectors, 418KB). Terminated postcodes only count where a sector has no live ones, so old addresses still resolve. The raw download is gitignored under `data/raw/`.  

### How outgoings are estimated

- ONS figures are average **weekly spend per household**. They are converted to monthly (`× 52 / 12`, rounded to pennies).
- One region is used per case (the figures are per household, not per person): that of the first client whose postcode resolves.
- Each outgoing is the sum of ONS category codes, defined in one table in `internal/outgoings/mapping.go`:

| Outgoing           | ONS codes                                                                       |
| ------------------ | ------------------------------------------------------------------------------- |
| water              | 4.3                                                                             |
| communications     | 8                                                                               |
| mortgage_rent      | 4.1.3 net rent, 14.5 purchase/alteration of dwellings, mortgages                |
| council_tax        | 13.1 (ONS publishes "mortgage interest payments, council tax etc." as one line) |
| insurance          | 12.4.1, 12.4.2, 12.4.4, 14.2                                                    |
| investments        | 14.6                                                                            |
| food               | 1                                                                               |
| clothing           | 3                                                                               |
| other_living_costs | 2, 5, 6, 10, 12.1, 12.2, 12.5, 13.2, 13.4                                       |
| entertainment      | 9.1, 9.2, 9.3.1, 9.3.2, 9.3.4, 9.3.5, 9.4.2, 9.4.4, 9.4.5, 9.4.6, 9.5, 11.1     |
| holidays           | 9.6, 11.2, 13.3                                                                 |
| sports             | 9.3.3, 9.4.1                                                                    |
| pension            | 14.1                                                                            |
| car_costs          | 7.2, 12.4.3                                                                     |
| other_transport_costs | 7.3                                                                          |
| child_care         | 12.3 social protection (includes childcare)                                     |
| fuel               | 4.4 electricity, gas, other fuels                                               |
| ground_rent_service_charge_shared_equity_rent | no ONS line → 0                                      |
| television_license | 9.4.3 (combined with TV subscriptions)                                          |
| household_repairs. | 4.2                                                                             |
| additional_details | text describing the source and region                                           |
  
Not used on purpose: 7.1 vehicle purchase (capital spend), 14.3/14.4 income tax and NI (deducted from income), 14.7 loan repayments, 14.8 gambling windfalls. A test checks that no code is counted twice (directly or through a parent/child code), and the server refuses to start if the mapping references a code missing from the data.

### Non-functional considerations

- **Fails fast at startup**: datasets, mapping and case file are loaded and validated before the server listens.
- **Performance**: all lookups are in-memory map reads; no per-request I/O beyond the case lookup.
- **Operability**: structured JSON request logs (`log/slog`), `/healthz`, read/write/idle timeouts, graceful shutdown on SIGINT/SIGTERM, panic recovery, internal errors never leaked to callers.
- **Maintainability**: small packages with interfaces at the boundaries (`CaseRepository`, `RegionResolver`, `CaseEstimator`) so parts can be swapped; mapping lives in one table; data converters are reproducible and separate from the server. Only `cmd/onsconvert` has an external dependency (`excelize`); the server uses the standard library only.
- **Testing**: unit tests for the xlsx parser (with rows shaped like the real sheet), postcode parsing/lookup, the estimator, and HTTP handlers via `httptest`.

### Layout

```

cmd/server HTTP server entry point
cmd/onsconvert one-off xlsx -> JSON converter (includes the xlsx row parser)
cmd/postcodeconvert one-off NSPL -> JSON converter (includes the sector/district table builder)
internal/api HTTP handlers and middleware
internal/cases Case model and repository (file-backed stand-in for the case service)
internal/ons ONS dataset model (+ data/ embedded JSON)
internal/outgoings ONS code mapping and estimator
internal/postcode postcode parsing and region lookup (+ data/ embedded JSON)

```
## Extension: Hybrid Values

I haven't implemented the extension. My notes on how I'd approach it are in [docs/hybrid-values.md](docs/hybrid-values.md).

## Disclaimer of AI usage

I confirm that I have used AI to assist with this take home task in the following ways and for the following reasons:

- Since the recommendation was in Golang, and my Golang is not very strong, I used AI to assist with syntax and how my design decisions could be implemented in Golang.
- Due to the fast turnaround for this task and commitments that were already planned before the take home task was assigned, I was tigher on time, and so used AI to speed up development in fleshing out ideas and code generation.
