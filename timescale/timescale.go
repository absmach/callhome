// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"fmt"
	"strings"

	"github.com/absmach/callhome"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

var _ callhome.TelemetryRepo = (*repo)(nil)

type repo struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) callhome.TelemetryRepo {
	return &repo{db: db}
}

// RetrieveAll gets all records from repo - optimized query.
func (r repo) RetrieveAll(ctx context.Context, pm callhome.PageMetadata, filters callhome.TelemetryFilters) (callhome.TelemetryPage, error) {
	filterQuery, params := generateQuery(filters)

	// Optimized query using DISTINCT ON for better performance
	// DISTINCT ON is much faster than ROW_NUMBER() window function
	q := fmt.Sprintf(`
	WITH latest_per_ip AS (
		SELECT DISTINCT ON (COALESCE(deployment_id, ip_address))
			ip_address,
			deployment_id,
			time,
			service_time,
			longitude,
			latitude,
			mg_version,
			country,
			city
		FROM telemetry
		%s
		ORDER BY COALESCE(deployment_id, ip_address), time DESC
	),
	limited_ips AS (
		SELECT *
		FROM latest_per_ip
		ORDER BY time DESC
		LIMIT :limit OFFSET :offset
	),
	services_per_ip AS (
		SELECT
			COALESCE(t.deployment_id, t.ip_address) as id,
			ARRAY_AGG(DISTINCT t.service) as services
		FROM limited_ips lpi
		INNER JOIN telemetry t ON COALESCE(t.deployment_id, t.ip_address) = COALESCE(lpi.deployment_id, lpi.ip_address)
		GROUP BY COALESCE(t.deployment_id, t.ip_address)
	)
	SELECT
		lpi.ip_address,
		lpi.time,
		lpi.service_time,
		lpi.longitude,
		lpi.latitude,
		lpi.mg_version,
		lpi.country,
		lpi.city,
		s.services
	FROM limited_ips lpi
	LEFT JOIN services_per_ip s ON COALESCE(lpi.deployment_id, lpi.ip_address) = s.id
	ORDER BY lpi.time DESC;
	`, filterQuery)

	params["limit"] = pm.Limit
	params["offset"] = pm.Offset

	rows, err := r.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return callhome.TelemetryPage{}, err
	}
	defer rows.Close()

	var results callhome.TelemetryPage

	for rows.Next() {
		var result callhome.Telemetry
		if err := rows.StructScan(&result); err != nil {
			return callhome.TelemetryPage{}, err
		}
		results.Telemetry = append(results.Telemetry, result)
	}

	// Set total to the number of results for simplicity
	// Since UI loads all data at once, exact count is less critical
	results.Total = uint64(len(results.Telemetry))

	return results, nil
}

// Save creates record in repo.
func (r repo) Save(ctx context.Context, t callhome.Telemetry) error {
	q := `INSERT INTO telemetry (ip_address, mac_address, deployment_id, longitude, latitude,
		mg_version, service, time, country, city, service_time)
		VALUES (:ip_address, :mac_address, :deployment_id, :longitude, :latitude,
			:mg_version, :service, :time, :country, :city, :service_time);`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(ErrSaveEvent, err.Error())
	}
	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(ErrTransRollback, txErr.Error()).Error())
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(ErrSaveEvent, err.Error())
		}
	}()

	if _, err := tx.NamedExec(q, t); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.InvalidTextRepresentation {
				return errors.Wrap(ErrSaveEvent, ErrInvalidEvent.Error())
			}
		}
		return errors.Wrap(ErrSaveEvent, err.Error())
	}
	return nil
}

// RetrieveSummary retrieve distinct - optimized to use single query.
func (r repo) RetrieveSummary(ctx context.Context, filters callhome.TelemetryFilters) (callhome.TelemetrySummary, error) {
	filterQuery, params := generateQuery(filters)
	var summary callhome.TelemetrySummary

	// Single optimized query that gets all distinct values and country counts at once
	q := fmt.Sprintf(`
		SELECT
			country,
			COUNT(DISTINCT COALESCE(deployment_id, ip_address)) as number_of_deployments,
			ARRAY_AGG(DISTINCT city) FILTER (WHERE city IS NOT NULL) as cities,
			ARRAY_AGG(DISTINCT service) FILTER (WHERE service IS NOT NULL) as services,
			ARRAY_AGG(DISTINCT mg_version) FILTER (WHERE mg_version IS NOT NULL) as versions
		FROM telemetry
		%s
		GROUP BY country;
	`, filterQuery)

	type queryResult struct {
		Country       string         `db:"country"`
		NoDeployments int            `db:"number_of_deployments"`
		Cities        pq.StringArray `db:"cities"`
		Services      pq.StringArray `db:"services"`
		Versions      pq.StringArray `db:"versions"`
	}

	rows, err := r.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return callhome.TelemetrySummary{}, err
	}
	defer rows.Close()

	citiesMap := make(map[string]bool)
	servicesMap := make(map[string]bool)
	versionsMap := make(map[string]bool)

	for rows.Next() {
		var result queryResult
		if err := rows.StructScan(&result); err != nil {
			return callhome.TelemetrySummary{}, err
		}

		summary.Countries = append(summary.Countries, callhome.CountrySummary{
			Country:       result.Country,
			NoDeployments: result.NoDeployments,
		})
		summary.TotalDeployments += result.NoDeployments

		// Collect unique cities, services, versions across all countries
		for _, city := range result.Cities {
			if city != "" {
				citiesMap[city] = true
			}
		}
		for _, service := range result.Services {
			if service != "" {
				servicesMap[service] = true
			}
		}
		for _, version := range result.Versions {
			if version != "" {
				versionsMap[version] = true
			}
		}
	}

	// Convert maps to slices
	for city := range citiesMap {
		summary.Cities = append(summary.Cities, city)
	}
	for service := range servicesMap {
		summary.Services = append(summary.Services, service)
	}
	for version := range versionsMap {
		summary.Versions = append(summary.Versions, version)
	}

	return summary, nil
}

func generateQuery(filters callhome.TelemetryFilters) (string, map[string]interface{}) {
	var queries []string
	params := make(map[string]interface{})

	if !filters.From.IsZero() {
		queries = append(queries, "time >= :from")
		params["from"] = filters.From
	}
	if !filters.To.IsZero() {
		queries = append(queries, "time <= :to")
		params["to"] = filters.To
	}
	if filters.Country != "" {
		queries = append(queries, "country = :country")
		params["country"] = filters.Country
	}

	if filters.City != "" {
		queries = append(queries, "city = :city")
		params["city"] = filters.City
	}

	if filters.Version != "" {
		queries = append(queries, "mg_version = :version")
		params["version"] = filters.Version
	}

	if filters.Service != "" {
		queries = append(queries, "service = :service")
		params["service"] = filters.Service
	}

	switch len(queries) {
	case 0:
		return "", params
	default:
		return fmt.Sprintf("WHERE %s", strings.Join(queries, " AND ")), params
	}
}
