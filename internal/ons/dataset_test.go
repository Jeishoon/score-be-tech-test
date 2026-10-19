package ons

import (
	"errors"
	"testing"
)

const sample = `{
  "source": "test", "year": 2019, "currency": "GBP",
  "regions": ["UK", "London"],
  "total_expenditure": {"UK": 500, "London": 600},
  "items": [
    {"code": "4.3", "label": "Water", "values": {"UK": 9.3, "London": 10.1}},
    {"code": "4.1.4", "label": "Second dwelling rent", "values": {"UK": 0.1, "London": null}}
  ]
}`

func TestValue(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	v, fellBack, err := d.Value("4.3", "London")
	if err != nil || v != 10.1 || fellBack {
		t.Errorf("Value(4.3, London) = %v, %v, %v", v, fellBack, err)
	}
	v, fellBack, err = d.Value("4.1.4", "London")
	if err != nil || v != 0.1 || !fellBack {
		t.Errorf("Value(4.1.4, London) = %v, %v, %v; want UK fallback 0.1", v, fellBack, err)
	}
	if _, _, err := d.Value("99", "London"); !errors.Is(err, ErrUnknownCode) {
		t.Errorf("unknown code err = %v", err)
	}
	if _, _, err := d.Value("4.3", "Atlantis"); !errors.Is(err, ErrUnknownRegion) {
		t.Errorf("unknown region err = %v", err)
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for name, in := range map[string]string{
		"not json":  `{`,
		"no items":  `{"regions":["UK"],"items":[]}`,
		"no UK":     `{"regions":["London"],"items":[{"code":"1","values":{}}]}`,
		"duplicate": `{"regions":["UK"],"items":[{"code":"1","values":{}},{"code":"1","values":{}}]}`,
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
