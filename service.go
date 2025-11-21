// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package callhome

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	pageLimit       = 1000
	summaryCacheTTL = 5 * time.Minute
)

// Service to receive homing telemetry data, persist and retrieve it.
type Service interface {
	// Save saves the homing telemetry data and its location information.
	Save(ctx context.Context, t Telemetry) error
	// Retrieve retrieves homing telemetry data from the specified repository.
	Retrieve(ctx context.Context, pm PageMetadata, filters TelemetryFilters) (TelemetryPage, error)
	// RetrieveSummary gets distinct countries and ip addresses
	RetrieveSummary(ctx context.Context, filters TelemetryFilters) (TelemetrySummary, error)
	// ServeUI gets the callhome index html page
	ServeUI(ctx context.Context, filters TelemetryFilters) ([]byte, error)
}

var _ Service = (*telemetryService)(nil)

type cachedSummary struct {
	summary   TelemetrySummary
	timestamp time.Time
}

type telemetryService struct {
	repo         TelemetryRepo
	locSvc       LocationService
	summaryCache *cachedSummary
	cacheMutex   sync.RWMutex
}

// New creates a new instance of the telemetry service.
func New(repo TelemetryRepo, locSvc LocationService) Service {
	return &telemetryService{
		repo:   repo,
		locSvc: locSvc,
	}
}

// Retrieve retrieves homing telemetry data from the specified repository.
func (ts *telemetryService) Retrieve(ctx context.Context, pm PageMetadata, filters TelemetryFilters) (TelemetryPage, error) {
	return ts.repo.RetrieveAll(ctx, pm, filters)
}

// Save saves the homing telemetry data and its location information.
func (ts *telemetryService) Save(ctx context.Context, t Telemetry) error {
	locRec, err := ts.locSvc.GetLocation(ctx, t.IpAddress)
	if err != nil {
		return err
	}
	t.City = locRec.City
	t.Country = locRec.Country_long
	t.Latitude = float64(locRec.Latitude)
	t.Longitude = float64(locRec.Longitude)
	t.LastSeen = time.Now()
	return ts.repo.Save(ctx, t)
}

func (ts *telemetryService) RetrieveSummary(ctx context.Context, filters TelemetryFilters) (TelemetrySummary, error) {
	return ts.repo.RetrieveSummary(ctx, filters)
}

// getCachedOrFetchSummary retrieves the unfiltered summary from cache if available and fresh,
// otherwise fetches it from the repository and updates the cache.
func (ts *telemetryService) getCachedOrFetchSummary(ctx context.Context) (TelemetrySummary, error) {
	// Try to read from cache first
	ts.cacheMutex.RLock()
	if ts.summaryCache != nil && time.Since(ts.summaryCache.timestamp) < summaryCacheTTL {
		cached := ts.summaryCache.summary
		ts.cacheMutex.RUnlock()
		return cached, nil
	}
	ts.cacheMutex.RUnlock()

	// Cache miss or expired, fetch from repository
	summary, err := ts.repo.RetrieveSummary(ctx, TelemetryFilters{})
	if err != nil {
		return TelemetrySummary{}, err
	}

	// Update cache
	ts.cacheMutex.Lock()
	ts.summaryCache = &cachedSummary{
		summary:   summary,
		timestamp: time.Now(),
	}
	ts.cacheMutex.Unlock()

	return summary, nil
}

// ServeUI gets the callhome index html page.
func (ts *telemetryService) ServeUI(ctx context.Context, filters TelemetryFilters) ([]byte, error) {
	tmpl := template.Must(template.ParseFiles("./web/template/index.html"))

	summary, err := ts.repo.RetrieveSummary(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Use cached unfiltered summary if available and fresh
	unfilteredSummary, err := ts.getCachedOrFetchSummary(ctx)
	if err != nil {
		return nil, err
	}

	telPage, err := ts.repo.RetrieveAll(ctx, PageMetadata{Limit: pageLimit}, filters)
	if err != nil {
		return nil, err
	}

	pg, err := json.Marshal(telPage)
	if err != nil {
		return nil, err
	}
	countries, err := json.Marshal(summary.Countries)
	if err != nil {
		return nil, err
	}

	var from, to string
	if !filters.From.IsZero() {
		from = filters.From.Format(time.RFC3339)
		from = strings.ReplaceAll(from, "Z", "")
	}
	if !filters.To.IsZero() {
		to = filters.To.Format(time.RFC3339)
		to = strings.ReplaceAll(to, "Z", "")
	}
	data := struct {
		Countries       string
		Cities          string
		FilterCountries []CountrySummary
		FilterCities    []string
		FilterServices  []string
		FilterVersions  []string
		NoDeployments   int
		NoCountries     int
		MapData         string
		From            string
		To              string
		SelectedCountry string
		SelectedCity    string
		SelectedService string
		SelectedVersion string
	}{
		Countries:       string(countries),
		FilterCountries: unfilteredSummary.Countries,
		FilterCities:    unfilteredSummary.Cities,
		FilterServices:  unfilteredSummary.Services,
		FilterVersions:  unfilteredSummary.Versions,
		NoDeployments:   summary.TotalDeployments,
		NoCountries:     len(summary.Countries),
		MapData:         string(pg),
		From:            from,
		To:              to,
		SelectedCountry: filters.Country,
		SelectedCity:    filters.City,
		SelectedService: filters.Service,
		SelectedVersion: filters.Version,
	}

	var res bytes.Buffer
	if err = tmpl.Execute(&res, data); err != nil {
		return nil, err
	}
	return res.Bytes(), nil
}
