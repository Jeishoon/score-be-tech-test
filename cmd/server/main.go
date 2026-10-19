// Command server runs the ONS outgoings API.
//
//	go run ./cmd/server
//	curl localhost:8080/cases/56763bf7-d046-4a54-a4a1-0728c724a092/outgoings
//
// Configuration (environment variables):
//
//	ADDR        listen address (default ":8080")
//	CASES_FILE  JSON file with one case or an array of cases (default "case.json")
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasonzou/score-be-tech-test/internal/api"
	"github.com/jasonzou/score-be-tech-test/internal/cases"
	"github.com/jasonzou/score-be-tech-test/internal/ons"
	"github.com/jasonzou/score-be-tech-test/internal/outgoings"
	"github.com/jasonzou/score-be-tech-test/internal/postcode"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := envOr("ADDR", ":8080")
	casesFile := envOr("CASES_FILE", "case.json")

	// Load everything up front so bad data stops the server at startup
	// rather than failing requests later.
	repo, err := cases.LoadFile(casesFile)
	if err != nil {
		return fmt.Errorf("load cases: %w", err)
	}
	ds, err := ons.Region2019()
	if err != nil {
		return fmt.Errorf("load ONS dataset: %w", err)
	}
	est, err := outgoings.NewEstimator(ds)
	if err != nil {
		return err
	}
	regions, err := postcode.LoadTable()
	if err != nil {
		return fmt.Errorf("load postcode table: %w", err)
	}
	log.Info("data loaded", "cases", repo.Len(), "ons_items", len(ds.Items),
		"postcode_sectors", len(regions.Sectors), "postcode_source", regions.Source)

	source := fmt.Sprintf("ONS Family Spending %d, expenditure by region (%s)", ds.Year, ds.Source)
	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(outgoings.NewService(repo, regions, est), source, log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// Let in-flight requests finish before exiting.
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
