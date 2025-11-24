// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package callhome

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/dgraph-io/ristretto"
)

const (
	pageLimit        = 1000
	summaryCacheTTL  = 5 * time.Minute
	cacheNumCounters = 1000      // Number of keys to track frequency (10x max items)
	cacheMaxCost     = 500 << 20 // 500MB max cache size
	cacheBufferItems = 64        // Number of keys per Get/Set buffer
	cacheCost        = 1 << 20   // Estimated cost per entry (~1MB)
	summaryCacheCost = 10 << 10  // Estimated cost per summary (~10KB)
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

type cachedTelemetryPage struct {
	page      TelemetryPage
	timestamp time.Time
}

type telemetryService struct {
	repo   TelemetryRepo
	locSvc LocationService
	cache  *ristretto.Cache
}

// New creates a new instance of the telemetry service.
func New(repo TelemetryRepo, locSvc LocationService) Service {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheNumCounters,
		MaxCost:     cacheMaxCost,
		BufferItems: cacheBufferItems,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create cache: %v", err))
	}

	return &telemetryService{
		repo:   repo,
		locSvc: locSvc,
		cache:  cache,
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

// getCachedOrFetchSummary retrieves summary from cache if available and fresh,
// otherwise fetches it from the repository and updates the cache.
// Thread-safe for concurrent access from multiple users using ristretto.
func (ts *telemetryService) getCachedOrFetchSummary(ctx context.Context, filters TelemetryFilters) (TelemetrySummary, error) {
	cacheKey := "summary:" + generateCacheKey(filters)

	// Try to read from cache first
	if val, found := ts.cache.Get(cacheKey); found {
		if cached, ok := val.(*cachedSummary); ok {
			if time.Since(cached.timestamp) < summaryCacheTTL {
				// Cache hit and still fresh
				return cached.summary, nil
			}
		}
	}

	// Cache miss or expired - fetch from repository
	summary, err := ts.repo.RetrieveSummary(ctx, filters)
	if err != nil {
		return TelemetrySummary{}, err
	}

	// Update cache
	ts.cache.Set(cacheKey, &cachedSummary{
		summary:   summary,
		timestamp: time.Now(),
	}, summaryCacheCost)
	ts.cache.Wait() // Wait for value to pass through buffers

	return summary, nil
}

// generateCacheKey creates a unique cache key from TelemetryFilters.
func generateCacheKey(filters TelemetryFilters) string {
	// Create a deterministic string representation of filters
	key := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		filters.Country,
		filters.City,
		filters.Service,
		filters.Version,
		filters.From.Format(time.RFC3339Nano),
		filters.To.Format(time.RFC3339Nano),
	)

	// Hash to keep keys short and uniform
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// getCachedOrFetchTelemetryPage retrieves telemetry page from cache if available and fresh,
// otherwise fetches it from the repository and updates the cache.
// Thread-safe for concurrent access from multiple users using ristretto.
func (ts *telemetryService) getCachedOrFetchTelemetryPage(ctx context.Context, filters TelemetryFilters) (TelemetryPage, error) {
	cacheKey := "page:" + generateCacheKey(filters)
	fmt.Println("cached key", cacheKey)

	// Try to read from cache first
	if val, found := ts.cache.Get(cacheKey); found {
		if cached, ok := val.(*cachedTelemetryPage); ok {
			if time.Since(cached.timestamp) < summaryCacheTTL {
				// Cache hit and still fresh
				return cached.page, nil
			}
		}
	}

	// Cache miss or expired - fetch from repository
	telPage, err := ts.repo.RetrieveAll(ctx, PageMetadata{Limit: pageLimit}, filters)
	if err != nil {
		return TelemetryPage{}, err
	}

	// Update cache
	ts.cache.Set(cacheKey, &cachedTelemetryPage{
		page:      telPage,
		timestamp: time.Now(),
	}, cacheCost)
	ts.cache.Wait() // Wait for value to pass through buffers

	return telPage, nil
}

// ServeUI gets the callhome index html page.
func (ts *telemetryService) ServeUI(ctx context.Context, filters TelemetryFilters) ([]byte, error) {
	tmpl := template.Must(template.ParseFiles("./web/template/index.html"))
	summary, err := ts.getCachedOrFetchSummary(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Use cached unfiltered summary for filter dropdowns
	unfilteredSummary, err := ts.getCachedOrFetchSummary(ctx, TelemetryFilters{})
	if err != nil {
		return nil, err
	}

	telPage, err := ts.getCachedOrFetchTelemetryPage(ctx, filters)
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
