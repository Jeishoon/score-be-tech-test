package postcode

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]struct{ sector, district string }{
		"NG1 2ET":   {"NG1 2", "NG1"},
		"ng12et":    {"NG1 2", "NG1"},
		" NG1  2ET": {"NG1 2", "NG1"},
		"SW1A 1AA":  {"SW1A 1", "SW1A"},
		"M60 1NW":   {"M60 1", "M60"},
		"EH1 1YZ":   {"EH1 1", "EH1"},
	}
	for in, want := range cases {
		pc, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if pc.Sector() != want.sector || pc.District() != want.district {
			t.Errorf("Parse(%q) = sector %q district %q, want %q %q", in, pc.Sector(), pc.District(), want.sector, want.district)
		}
	}
	for _, in := range []string{"", "NOTAPOSTCODE", "12345", "NG1", "NG1 2E"} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalidPostcode) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalidPostcode", in, err)
		}
	}
}
