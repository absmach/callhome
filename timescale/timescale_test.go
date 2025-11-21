// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/absmach/callhome"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestSave(t *testing.T) {
	ctx := context.TODO()
	mockTelemetry := callhome.Telemetry{
		Services:    []string{},
		Service:     "mock service",
		Longitude:   1.2,
		Latitude:    30.2,
		IpAddress:   "192.168.0.1",
		Version:     "0.13",
		LastSeen:    time.Now(),
		Country:     "someCountry",
		City:        "someCity",
		ServiceTime: time.Now(),
	}
	t.Run("failed to start transactions", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()

		assert.Nil(t, err)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("eny error"))

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		err = repo.Save(ctx, mockTelemetry)
		assert.NotNil(t, err)
	})
	t.Run("failed exec", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		mock.ExpectBegin()

		mock.ExpectExec("INSERT INTO telemetry").WillReturnError(fmt.Errorf("failed save"))

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		err = repo.Save(ctx, mockTelemetry)
		assert.NotNil(t, err)
	})
	t.Run("invalid text representation", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		mock.ExpectBegin()

		pgerr := pgconn.PgError{
			Code: pgerrcode.InvalidTextRepresentation,
		}

		mock.ExpectExec("INSERT INTO telemetry").WillReturnError(&pgerr)

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		err = repo.Save(ctx, mockTelemetry)
		assert.NotNil(t, err)
	})
	t.Run("successful save", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		mock.ExpectBegin()

		mock.ExpectExec("INSERT INTO telemetry").WillReturnResult(sqlmock.NewResult(0, 1))

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		err = repo.Save(ctx, mockTelemetry)
		assert.Nil(t, err)
	})
}

func TestRetrieveAll(t *testing.T) {
	ctx := context.TODO()
	now := time.Now()
	mTel := callhome.Telemetry{
		Service:     "mock service",
		Longitude:   1.2,
		Latitude:    30.2,
		IpAddress:   "192.168.0.1",
		Version:     "0.13",
		LastSeen:    now,
		Country:     "someCountry",
		City:        "someCity",
		ServiceTime: now,
	}
	t.Run("error performing select", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		mock.ExpectQuery("SELECT(.*)").WillReturnError(fmt.Errorf("any error"))

		_, err = repo.RetrieveAll(ctx, callhome.PageMetadata{Limit: 10, Offset: 0}, callhome.TelemetryFilters{})
		assert.NotNil(t, err)
	})
	t.Run("successful", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		services := pq.Array([]string{mTel.Service})
		rows := sqlmock.NewRows(
			[]string{"ip_address", "services", "time", "service_time", "longitude", "latitude", "mg_version", "country", "city"},
		).AddRow(mTel.IpAddress, services, mTel.LastSeen, mTel.ServiceTime, mTel.Longitude, mTel.Latitude, mTel.Version, mTel.Country, mTel.City)

		mock.ExpectQuery("WITH latest_telemetry(.*)").WillReturnRows(rows)

		tp, err := repo.RetrieveAll(ctx, callhome.PageMetadata{Limit: 10, Offset: 0}, callhome.TelemetryFilters{})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(tp.Telemetry))
		assert.Equal(t, mTel.IpAddress, tp.Telemetry[0].IpAddress)
		assert.Equal(t, mTel.Country, tp.Telemetry[0].Country)
		assert.Equal(t, uint64(1), tp.Total)
	})
}

func TestRetrieveSummary(t *testing.T) {
	ctx := context.TODO()
	t.Run("error performing query", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New()
		assert.Nil(t, err)

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		mock.ExpectQuery("SELECT(.*)").WillReturnError(fmt.Errorf("query error"))

		_, err = repo.RetrieveSummary(ctx, callhome.TelemetryFilters{})
		assert.NotNil(t, err)
	})

	// Note: sqlmock has limitations with PostgreSQL array types
	// Full array handling requires integration tests with real database
	// This test just verifies the query executes without error
	t.Run("successful summary retrieval - query validation", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		assert.Nil(t, err)

		defer sqlDB.Close()
		sqlxDB := sqlx.NewDb(sqlDB, "sqlmock")

		repo := New(sqlxDB)

		// Return empty result set to just verify query structure
		rows := sqlmock.NewRows(
			[]string{"country", "number_of_deployments", "cities", "services", "versions"},
		)

		mock.ExpectQuery("SELECT.*country.*number_of_deployments.*cities.*services.*versions.*FROM telemetry.*GROUP BY country").WillReturnRows(rows)

		summary, err := repo.RetrieveSummary(ctx, callhome.TelemetryFilters{})
		assert.Nil(t, err)
		assert.NotNil(t, summary)
		assert.Nil(t, mock.ExpectationsWereMet())
	})
}
