package cases

import (
	"context"
	"errors"
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

func TestNewMemoryRepositoryRejectsBadCases(t *testing.T) {
	if _, err := NewMemoryRepository(Case{}); err == nil {
		t.Error("want error for empty case_id")
	}
	if _, err := NewMemoryRepository(Case{CaseID: "a"}, Case{CaseID: "a"}); err == nil {
		t.Error("want error for duplicate case_id")
	}
}
