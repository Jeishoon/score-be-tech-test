package cases

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestLoadFileSingleCase(t *testing.T) {
	r, err := LoadFile("../../case.json")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	c, err := r.Get(context.Background(), "56763bf7-d046-4a54-a4a1-0728c724a092")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(c.Clients) != 2 || c.Clients[0].Details.Address.Postcode != "NG1 2ET" {
		t.Errorf("unexpected case: %+v", c)
	}
}

// TestCaseFileUniqueIDs guards against duplicate case IDs in case.json, which
// LoadFile would otherwise silently collapse into one entry.
func TestCaseFileUniqueIDs(t *testing.T) {
	data, err := os.ReadFile("../../case.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("case.json should be an array of cases: %v", err)
	}
	r, err := LoadFile("../../case.json")
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != len(raw) {
		t.Errorf("loaded %d unique case IDs from %d cases; case_id values must be unique", r.Len(), len(raw))
	}
}

func TestLoadFileArray(t *testing.T) {
	r, err := LoadFile("testdata/cases.json")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if r.Len() != 3 {
		t.Errorf("Len = %d, want 3", r.Len())
	}
	if _, err := r.Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) err = %v, want ErrNotFound", err)
	}
}
