// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/absmach/callhome"
	"github.com/stretchr/testify/mock"
)

var _ callhome.TelemetryRepo = (*mockRepo)(nil)

type mockRepo struct {
	mock.Mock
}

func (mr *mockRepo) RetrieveAll(ctx context.Context, pm callhome.PageMetadata, filter callhome.TelemetryFilters) (callhome.TelemetryPage, error) {
	ret := mr.Called(ctx, pm)
	return ret.Get(0).(callhome.TelemetryPage), ret.Error(1)
}

func (mr *mockRepo) Save(ctx context.Context, t callhome.Telemetry) error {
	ret := mr.Called(ctx, t)
	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, callhome.Telemetry) error); ok {
		r0 = rf(ctx, t)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (mr *mockRepo) RetrieveSummary(ctx context.Context, filter callhome.TelemetryFilters) (callhome.TelemetrySummary, error) {
	ret := mr.Called(ctx, filter)
	return ret.Get(0).(callhome.TelemetrySummary), ret.Error(1)
}

type mockConstructorTestingTNewTelemetryRepo interface {
	mock.TestingT
	Cleanup(func())
}

func NewTelemetryRepo(t mockConstructorTestingTNewTelemetryRepo) *mockRepo {
	mock := &mockRepo{}
	mock.Mock.Test(t)

	return mock
}
