package cases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MemoryRepository serves cases held in memory, typically loaded from a JSON
// file at startup with LoadFile. It stands in for the case service; in
// production this would be an HTTP client with the same Get method.
type MemoryRepository struct {
	cases map[string]Case
}

// LoadFile reads a JSON file holding either one case object or an array of
// cases.
func LoadFile(path string) (*MemoryRepository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list []Case
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		err = json.Unmarshal(data, &list)
	} else {
		var c Case
		err = json.Unmarshal(data, &c)
		list = []Case{c}
	}
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return NewMemoryRepository(list...)
}

// NewMemoryRepository builds a repository from cases already in memory.
func NewMemoryRepository(list ...Case) (*MemoryRepository, error) {
	r := &MemoryRepository{cases: make(map[string]Case, len(list))}
	for _, c := range list {
		if c.CaseID == "" {
			return nil, errors.New("case without case_id")
		}
		if _, dup := r.cases[c.CaseID]; dup {
			return nil, fmt.Errorf("duplicate case_id %q", c.CaseID)
		}
		r.cases[c.CaseID] = c
	}
	return r, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (Case, error) {
	c, ok := r.cases[id]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return c, nil
}

// Len returns the number of cases held.
func (r *MemoryRepository) Len() int { return len(r.cases) }
