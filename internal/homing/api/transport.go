package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/callhome/internal/homing"
	"github.com/mainflux/callhome/internal/homing/repository"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType  = "application/json"
	offsetKey    = "offset"
	limitKey     = "limit"
	ipAddressKey = "ip"
	metadataKey  = "metadata"
	statusKey    = "status"
	defOffset    = 0
	defLimit     = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc homing.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(LoggingErrorEncoder(logger, encodeError)),
	}

	mux := bone.New()

	mux.Post("/telemetry", kithttp.NewServer(
		kitot.TraceServer(tracer, "save")(saveEndpoint(svc)),
		decodeSaveTelemetryReq,
		encodeResponse,
		opts...,
	))

	mux.Get("/telemetry/:repo", kithttp.NewServer(
		kitot.TraceServer(tracer, "get all")(getAllEndpoint(svc)),
		decodeGetAll,
		encodeResponse,
		opts...,
	))
	mux.GetFunc("/health", mainflux.Health("telemetry"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(mainflux.Response); ok {
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
	case errors.Contains(err, errors.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity),
		err == ErrLimitSize,
		err == ErrOffsetSize:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization),
		errors.Contains(err, repository.ErrInvalidEvent):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	case errors.Contains(err, repository.ErrSaveEvent),
		errors.Contains(err, repository.ErrTransRollback):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func decodeGetAll(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}
	ip, err := ReadStringQuery(r, ipAddressKey, "")
	if err != nil {
		return nil, err
	}

	repo := bone.GetValue(r, "repo")
	if repo == "" {
		repo = homing.SheetsRepo
	}

	req := listTelemetryReq{
		offset:    o,
		limit:     l,
		IpAddress: ip,
		repo:      repo,
	}
	return req, nil
}

func decodeSaveTelemetryReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var telemetry telemetryReq
	if err := json.NewDecoder(r.Body).Decode(&telemetry); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return telemetry, nil
}
