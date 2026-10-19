// Command postcodeconvert builds the postcode -> ONS region table embedded by
// the server, from the ONS National Statistics Postcode Lookup (NSPL).
//
// Download the NSPL zip from the ONS Open Geography Portal
// (https://geoportal.statistics.gov.uk, search "National Statistics Postcode
// Lookup") and run once:
//
//	go run ./cmd/postcodeconvert -in data/raw/nspl.zip -out internal/postcode/data/sectors.json
//
// -in may be the zip itself or the extracted Data/NSPL_*_UK.csv.
package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

// The main all-UK CSV inside the zip, e.g. "Data/NSPL_AUG_2026_UK.csv".
var mainCSVRe = regexp.MustCompile(`^NSPL_.*_UK\.csv$`)

func main() {
	in := flag.String("in", "data/raw/nspl.zip", "NSPL zip or CSV")
	out := flag.String("out", "internal/postcode/data/sectors.json", "output JSON file")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		log.Fatal(err)
	}
}

func run(in, out string) error {
	r, name, closeFn, err := openCSV(in)
	if err != nil {
		return err
	}
	defer closeFn()

	b := postcode.NewBuilder()
	rows, err := readNSPL(r, b)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}

	tbl, st := b.Table(name)
	data, err := json.MarshalIndent(tbl, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("read %d postcodes (%d skipped): wrote %s with %d sectors (%d span regions), %d districts\n",
		rows, b.Skipped, out, st.Sectors, st.MixedSectors, st.Districts)
	return nil
}

// readNSPL streams the CSV into b and returns the number of data rows.
// Columns are found by name because NSPL suffixes them with the boundary
// year (e.g. "rgn26cd"), which changes between releases.
func readNSPL(r io.Reader, b *postcode.Builder) (int, error) {
	cr := csv.NewReader(r)
	cr.ReuseRecord = true
	header, err := cr.Read()
	if err != nil {
		return 0, fmt.Errorf("header: %w", err)
	}
	pcCol, termCol, rgnCol := -1, -1, -1
	for i, h := range header {
		h = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
		switch {
		case h == "pcds":
			pcCol = i
		case h == "doterm":
			termCol = i
		case strings.HasPrefix(h, "rgn") && strings.HasSuffix(h, "cd"), h == "rgn":
			rgnCol = i
		}
	}
	if pcCol < 0 || termCol < 0 || rgnCol < 0 {
		return 0, fmt.Errorf("missing pcds/doterm/rgn columns in header %v", header)
	}

	n := 0
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return n, nil
		}
		if err != nil {
			return n, err
		}
		n++
		b.Add(rec[pcCol], rec[rgnCol], strings.TrimSpace(rec[termCol]) != "")
	}
}

func openCSV(in string) (io.Reader, string, func(), error) {
	if !strings.EqualFold(filepath.Ext(in), ".zip") {
		f, err := os.Open(in)
		if err != nil {
			return nil, "", nil, err
		}
		return f, filepath.Base(in), func() { f.Close() }, nil
	}

	zr, err := zip.OpenReader(in)
	if err != nil {
		return nil, "", nil, err
	}
	for _, f := range zr.File {
		if mainCSVRe.MatchString(path.Base(f.Name)) {
			rc, err := f.Open()
			if err != nil {
				zr.Close()
				return nil, "", nil, err
			}
			return rc, path.Base(f.Name), func() { rc.Close(); zr.Close() }, nil
		}
	}
	zr.Close()
	return nil, "", nil, fmt.Errorf("no NSPL_*_UK.csv found in %s", in)
}
