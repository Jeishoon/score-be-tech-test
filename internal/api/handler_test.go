package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jasonzou/score-be-tech-test/internal/cases"
	"github.com/jasonzou/score-be-tech-test/internal/ons"
	"github.com/jasonzou/score-be-tech-test/internal/outgoings"
	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

const sampleCaseID = "56763bf7-d046-4a54-a4a1-0728c724a092"

// newServer wires the real embedded data with the case fixtures (the sample
// case.json plus a London case and an invalid-postcode case).
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo, err := cases.LoadFile("../cases/testdata/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	ds, err := ons.Region2019()
	if err != nil {
		t.Fatal(err)
	}
	est, err := outgoings.NewEstimator(ds)
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := postcode.LoadTable()
	if err != nil {
		t.Fatal(err)
	}
	svc := outgoings.NewService(repo, tbl, est)
	h := New(svc, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, url string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	return resp.StatusCode, body
}

func TestOutgoingsSampleCase(t *testing.T) {
	srv := newServer(t)
	status, body := get(t, srv.URL+"/cases/"+sampleCaseID+"/outgoings")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body %s", status, body)
	}
	var resp OutgoingsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Region != "East Midlands" || resp.Period != "monthly" || resp.Postcode != "NG1 2ET" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if resp.Outgoings.Food != 265.20 {
		t.Errorf("food = %v, want 265.20", resp.Outgoings.Food)
	}
	// Every documented field name must be present in the JSON.
	for _, f := range []string{"outgoings_water", "outgoings_ground_rent_service_charge_shared_equity_rent", "outgoings_additional_details"} {
		if !strings.Contains(string(body), `"`+f+`"`) {
			t.Errorf("response missing %s", f)
		}
	}
}

func TestOutgoingsErrors(t *testing.T) {
	srv := newServer(t)
	cases := map[string]int{
		"/cases/not-a-uuid/outgoings":                           http.StatusNotFound,
		"/cases/00000000-0000-4000-8000-000000000000/outgoings": http.StatusNotFound,
		"/cases/22222222-2222-4222-8222-222222222222/outgoings": http.StatusUnprocessableEntity,
		"/nope": http.StatusNotFound,
	}
	for path, want := range cases {
		status, body := get(t, srv.URL+path)
		if status != want {
			t.Errorf("GET %s = %d, want %d (body %s)", path, status, want, body)
		}
	}
}

func TestOutgoingsLondonCase(t *testing.T) {
	srv := newServer(t)
	status, body := get(t, srv.URL+"/cases/11111111-1111-4111-8111-111111111111/outgoings")
	if status != http.StatusOK || !strings.Contains(string(body), `"region":"London"`) {
		t.Errorf("status = %d, body %s", status, body)
	}
}

func TestHealth(t *testing.T) {
	srv := newServer(t)
	if status, _ := get(t, srv.URL+"/healthz"); status != http.StatusOK {
		t.Errorf("status = %d", status)
	}
}

func TestCorsAllowsViteOriginAndPreflight(t *testing.T) {
	h := New(failingService{}, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodOptions, "/cases/"+sampleCaseID+"/outgoings", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("preflight status=%d headers=%v", rec.Code, rec.Header())
	}
}

func TestCorsDoesNotAllowUnknownOrigin(t *testing.T) {
	h := New(failingService{}, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://untrusted.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected allow-origin header: %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

type failingService struct{}

func (failingService) EstimateCase(context.Context, string) (outgoings.CaseEstimate, error) {
	return outgoings.CaseEstimate{}, errors.New("db down")
}

func TestInternalErrorHidesDetails(t *testing.T) {
	h := New(failingService{}, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/cases/"+sampleCaseID+"/outgoings", nil))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Errorf("status = %d, body %s", rec.Code, rec.Body)
	}
}
