// Command onsconvert converts an ONS expenditure-by-region xlsx into the JSON
// file embedded by the server. Run it once when the source data changes:
//
//	go run ./cmd/onsconvert -in ONSByRegion2019.xlsx -out internal/ons/data/region2019.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"github.com/jasonzou/score-be-tech-test/internal/ons"
	"github.com/jasonzou/score-be-tech-test/internal/onsimport"
)

func main() {
	in := flag.String("in", "ONSByRegion2019.xlsx", "source xlsx file")
	out := flag.String("out", "internal/ons/data/region2019.json", "output JSON file")
	year := flag.Int("year", 2019, "survey year")
	flag.Parse()

	if err := run(*in, *out, *year); err != nil {
		log.Fatal(err)
	}
}

func run(in, out string, year int) error {
	f, err := excelize.OpenFile(in)
	if err != nil {
		return fmt.Errorf("open %s: %w", in, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("%s has no sheets", in)
	}
	// RawCellValue: read the stored numbers, not the display format.
	rows, err := f.GetRows(sheets[0], excelize.Options{RawCellValue: true})
	if err != nil {
		return fmt.Errorf("read rows: %w", err)
	}

	d, err := onsimport.Parse(rows, onsimport.Options{
		Source:   filepath.Base(in),
		Year:     year,
		Currency: "GBP",
	})
	if err != nil {
		return fmt.Errorf("parse %s: %w", in, err)
	}

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	// Round-trip through the server's loader so a bad file never gets written.
	if _, err := ons.Parse(data); err != nil {
		return fmt.Errorf("generated dataset is invalid: %w", err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d items, %d regions\n", out, len(d.Items), len(d.Regions))
	return nil
}
