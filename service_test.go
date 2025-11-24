// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package callhome_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/callhome"
	"github.com/absmach/callhome/mocks"
	"github.com/absmach/callhome/timescale"
	repoMocks "github.com/absmach/callhome/timescale/mocks"
	"github.com/ip2location/ip2location-go/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRetrieve(t *testing.T) {
	ctx := context.TODO()
	t.Run("failed repo save", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock all calls - background prefetch and test calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, timescale.ErrSaveEvent)
		svc := callhome.New(timescaleRepo, nil)
		_, err := svc.Retrieve(ctx, callhome.PageMetadata{}, callhome.TelemetryFilters{})
		assert.NotNil(t, err)
		assert.Equal(t, timescale.ErrSaveEvent, err)
	})
	t.Run("success", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock all calls - background prefetch and test calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		svc := callhome.New(timescaleRepo, nil)
		_, err := svc.Retrieve(ctx, callhome.PageMetadata{}, callhome.TelemetryFilters{})
		assert.Nil(t, err)
	})
}

func TestSave(t *testing.T) {
	ctx := context.TODO()
	t.Run("error obtaining location", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock the background prefetch calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		locMock := mocks.NewLocationService(t)
		locMock.On("GetLocation", mock.Anything, "").Return(ip2location.IP2Locationrecord{}, fmt.Errorf("error getting loc"))
		svc := callhome.New(timescaleRepo, locMock)
		err := svc.Save(ctx, callhome.Telemetry{})
		assert.NotNil(t, err)
	})
	t.Run("error saving to timescale", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock the background prefetch calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		locMock := mocks.NewLocationService(t)
		locMock.On("GetLocation", mock.Anything, "").Return(ip2location.IP2Locationrecord{
			Latitude:     1.2,
			Longitude:    30,
			Country_long: "SomeCountry",
			City:         "someCity",
		}, nil)
		timescaleRepo.On("Save", ctx, mock.AnythingOfType("callhome.Telemetry")).Return(timescale.ErrSaveEvent)
		svc := callhome.New(ctx, timescaleRepo, locMock)
		err := svc.Save(ctx, callhome.Telemetry{})
		assert.NotNil(t, err)
		assert.Equal(t, timescale.ErrSaveEvent, err)
	})
	t.Run("successful save", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock the background prefetch calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		locMock := mocks.NewLocationService(t)
		locMock.On("GetLocation", mock.Anything, "").Return(ip2location.IP2Locationrecord{
			Latitude:     1.2,
			Longitude:    30,
			Country_long: "SomeCountry",
			City:         "someCity",
		}, nil)
		timescaleRepo.On("Save", ctx, mock.AnythingOfType("callhome.Telemetry")).Return(nil)
		svc := callhome.New(ctx, timescaleRepo, locMock)
		err := svc.Save(ctx, callhome.Telemetry{})
		assert.Nil(t, err)
	})
	t.Run("successful update", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock the background prefetch calls
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		locMock := mocks.NewLocationService(t)
		locMock.On("GetLocation", mock.Anything, "").Return(ip2location.IP2Locationrecord{
			Latitude:     1.2,
			Longitude:    30,
			Country_long: "SomeCountry",
			City:         "someCity",
		}, nil)
		timescaleRepo.On("Save", ctx, mock.AnythingOfType("callhome.Telemetry")).Return(nil)
		svc := callhome.New(ctx, timescaleRepo, locMock)
		err := svc.Save(ctx, callhome.Telemetry{})
		assert.Nil(t, err)
	})
}

func TestRetrieveSummary(t *testing.T) {
	ctx := context.TODO()
	mockSummary := callhome.TelemetrySummary{
		Countries: []callhome.CountrySummary{
			{Country: "TestCountry", NoDeployments: 5},
		},
		Cities:           []string{"TestCity"},
		Services:         []string{"TestService"},
		Versions:         []string{"v1.0"},
		TotalDeployments: 5,
	}

	t.Run("failed repo retrieve", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock all calls - background prefetch and test calls will return error
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(callhome.TelemetrySummary{}, timescale.ErrSaveEvent)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		svc := callhome.New(timescaleRepo, nil)
		_, err := svc.RetrieveSummary(ctx, callhome.TelemetryFilters{})
		assert.NotNil(t, err)
		assert.Equal(t, timescale.ErrSaveEvent, err)
	})

	t.Run("success", func(t *testing.T) {
		timescaleRepo := repoMocks.NewTelemetryRepo(t)
		// Mock all calls - background prefetch and test calls will return mockSummary
		timescaleRepo.On("RetrieveSummary", mock.Anything, mock.Anything).Return(mockSummary, nil)
		timescaleRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything).Return(callhome.TelemetryPage{}, nil)
		svc := callhome.New(timescaleRepo, nil)
		summary, err := svc.RetrieveSummary(ctx, callhome.TelemetryFilters{})
		assert.Nil(t, err)
		assert.Equal(t, mockSummary, summary)
	})
}
