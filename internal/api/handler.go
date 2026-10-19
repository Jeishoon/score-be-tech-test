// Package api exposes the outgoings estimate over HTTP.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jasonzou/score-be-tech-test/internal/cases"
	"github.com/jasonzou/score-be-tech-test/internal/outgoings"
)

// CaseEstimator is the business logic the API exposes; an interface so tests
// can swap in a fake.
type CaseEstimator interface {
	EstimateCase(ctx context.Context, caseID string) (outgoings.CaseEstimate, error)
}

// Handler serves the API. It only translates between HTTP and the
// CaseEstimator; business rules live in package outgoings.
type Handler struct {
	svc    CaseEstimator
	source string
	log    *slog.Logger
}

// New returns the API's http.Handler with request logging.
// source describes the dataset in responses, e.g. "ONS Family Spending 2019".
func New(svc CaseEstimator, source string, log *slog.Logger) http.Handler {
	h := &Handler{svc: svc, source: source, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /cases/{caseID}/outgoings", h.outgoings)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return logRequests(log, mux)
}

// OutgoingsResponse is the body of GET /cases/{caseID}/outgoings.
type OutgoingsResponse struct {
	CaseID          string              `json:"case_id"`
	Region          string              `json:"region"`
	Postcode        string              `json:"postcode"`
	Period          string              `json:"period"`
	Currency        string              `json:"currency"`
	Source          string              `json:"source"`
	UKFallbackCodes []string            `json:"uk_fallback_codes,omitempty"`
	Outgoings       outgoings.Outgoings `json:"outgoings"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) outgoings(w http.ResponseWriter, r *http.Request) {
	est, err := h.svc.EstimateCase(r.Context(), r.PathValue("caseID"))
	switch {
	case errors.Is(err, cases.ErrNotFound):
		writeError(w, http.StatusNotFound, "case not found")
		return
	case errors.Is(err, outgoings.ErrNoRegion):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	case err != nil:
		h.internalError(w, r, "estimate outgoings", err)
		return
	}

	writeJSON(w, http.StatusOK, OutgoingsResponse{
		CaseID:          est.CaseID,
		Region:          est.Region,
		Postcode:        est.Postcode,
		Period:          "monthly",
		Currency:        "GBP",
		Source:          h.source,
		UKFallbackCodes: est.UKFallbackCodes,
		Outgoings:       est.Outgoings,
	})
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.log.ErrorContext(r.Context(), op+" failed", "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
