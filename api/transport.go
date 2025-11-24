// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/callhome"
	"github.com/absmach/callhome/timescale"
	"github.com/go-chi/chi"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	fromKey     = "from"
	toKey       = "to"
	countryKey  = "country"
	cityKey     = "city"
	versionKey  = "version"
	serviceKey  = "service"
	defOffset   = 0
	defLimit    = 10
	staticDir   = "./web/static"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc callhome.Service, tp trace.TracerProvider, logger *slog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(LoggingErrorEncoder(logger, encodeError)),
	}

	mux := chi.NewRouter()

	mux.Post("/telemetry",
		otelhttp.NewHandler(kithttp.NewServer(
			saveEndpoint(svc),
			decodeSaveTelemetryReq,
			encodeResponse,
			opts...,
		), "save").ServeHTTP)

	mux.Get("/telemetry",
		otelhttp.NewHandler(kithttp.NewServer(
			retrieveEndpoint(svc),
			decodeRetrieve,
			encodeResponse,
			opts...,
		), "retrieve").ServeHTTP)

	mux.Get("/telemetry/summary",
		otelhttp.NewHandler(kithttp.NewServer(
			retrieveSummaryEndpoint(svc),
			decodeRetrieve,
			encodeResponse,
			opts...,
		), "retrieve-summary").ServeHTTP)

	mux.Get("/",
		otelhttp.NewHandler(kithttp.NewServer(
			serveUI(svc),
			decodeRetrieve,
			encodeStaticResponse,
			opts...,
		), "serve-ui").ServeHTTP)

	mux.Get("/health", callhome.Health("home", "telemetry"))
	mux.Handle("/metrics", promhttp.Handler())

	// Static file handler
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/*", fs)

	return mux
}

func encodeStaticResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "text/html")
	ar, ok := response.(uiRes)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	for k, v := range ar.Headers() {
		w.Header().Set(k, v)
	}
	w.WriteHeader(ar.Code())

	if ar.Empty() {
		return nil
	}
	_, err := w.Write(ar.html)
	if err != nil {
		return err
	}
	return nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case
		errors.Is(err, ErrInvalidQueryParams),
		errors.Is(err, ErrMalformedEntity),
		err == ErrLimitSize,
		err == ErrOffsetSize:
		w.WriteHeader(http.StatusBadRequest)
	case err == ErrUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Is(err, timescale.ErrInvalidEvent):
		w.WriteHeader(http.StatusForbidden)
	case errors.Is(err, timescale.ErrSaveEvent),
		errors.Is(err, timescale.ErrTransRollback):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func decodeRetrieve(_ context.Context, r *http.Request) (interface{}, error) {
	t := time.Now().UTC()
	o, err := ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	fromString, err := ReadStringQuery(r, fromKey, "")
	if err != nil {
		return nil, err
	}

	toString, err := ReadStringQuery(r, toKey, "")
	if err != nil {
		return nil, err
	}

	var from, to time.Time
	if fromString != "" {
		from, err = time.Parse(time.RFC3339, fromString)
		if err != nil {
			return nil, err
		}
	} else {
		from = t.AddDate(0, 0, -30).Round(callhome.RoundPeriod)
	}
	if toString != "" {
		to, err = time.Parse(time.RFC3339, toString)
		if err != nil {
			return nil, err
		}
	} else {
		to = t.Round(callhome.RoundPeriod)
	}
	co, err := ReadStringQuery(r, countryKey, "")
	if err != nil {
		return nil, err
	}

	ci, err := ReadStringQuery(r, cityKey, "")
	if err != nil {
		return nil, err
	}

	ve, err := ReadStringQuery(r, versionKey, "")
	if err != nil {
		return nil, err
	}

	se, err := ReadStringQuery(r, serviceKey, "")
	if err != nil {
		return nil, err
	}

	req := listTelemetryReq{
		offset:  o,
		limit:   l,
		from:    from,
		to:      to,
		country: co,
		city:    ci,
		version: ve,
		service: se,
	}
	return req, nil
}

func decodeSaveTelemetryReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var telemetry saveTelemetryReq
	if err := json.NewDecoder(r.Body).Decode(&telemetry); err != nil {
		return nil, errors.Join(ErrMalformedEntity, err)
	}

	return telemetry, nil
}
