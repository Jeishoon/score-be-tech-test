.PHONY: run build test vet convert-ons convert-postcodes

NSPL ?= data/raw/nspl.zip

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

# One-off: regenerate internal/ons/data/region2019.json from the xlsx.
convert-ons:
	go run ./cmd/onsconvert -in ONSByRegion2019.xlsx -out internal/ons/data/region2019.json

# One-off: regenerate internal/postcode/data/sectors.json from the NSPL download.
convert-postcodes:
	go run ./cmd/postcodeconvert -in $(NSPL) -out internal/postcode/data/sectors.json
