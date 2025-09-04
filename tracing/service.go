// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/callhome"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	retrieveOp        = "retrieve_op"
	retrieveSummaryOp = "retrieve_summary_op"
	saveOp            = "save_op"
	serveUIOp         = "serve_UI_op"
)

var _ callhome.Service = (*telemetryServiceTracer)(nil)

type telemetryServiceTracer struct {
	tracer trace.Tracer
	svc    callhome.Service
}

// NewService adds tracing middleware to callhome.Service.
func NewService(tracer trace.Tracer, svc callhome.Service) callhome.Service {
	return &telemetryServiceTracer{
		tracer: tracer,
		svc:    svc,
	}
}

// Retrieve adds tracing middleware to retrieve method.
func (tst *telemetryServiceTracer) Retrieve(ctx context.Context, pm callhome.PageMetadata, filters callhome.TelemetryFilters) (callhome.TelemetryPage, error) {
	ctx, span := tst.tracer.Start(ctx, retrieveOp)
	defer span.End()
	return tst.svc.Retrieve(ctx, pm, filters)
}

// RetrieveSummary adds tracing middleware to RetrieveSummary.
func (tst *telemetryServiceTracer) RetrieveSummary(ctx context.Context, filters callhome.TelemetryFilters) (callhome.TelemetrySummary, error) {
	ctx, span := tst.tracer.Start(ctx, retrieveSummaryOp)
	defer span.End()
	return tst.svc.RetrieveSummary(ctx, filters)
}

// Save adds tracing middleware to Save.
func (tst *telemetryServiceTracer) Save(ctx context.Context, t callhome.Telemetry) error {
	ctx, span := tst.tracer.Start(ctx, saveOp, trace.WithAttributes(
		[]attribute.KeyValue{
			attribute.String("ip_address", t.IpAddress),
			attribute.String("mac_address", t.MacAddress),
		}...))
	defer span.End()
	return tst.svc.Save(ctx, t)
}

// ServeUI adds tracing middleware to ServeUI.
func (tst *telemetryServiceTracer) ServeUI(ctx context.Context, filters callhome.TelemetryFilters) ([]byte, error) {
	ctx, span := tst.tracer.Start(ctx, serveUIOp)
	defer span.End()
	return tst.svc.ServeUI(ctx, filters)
}
