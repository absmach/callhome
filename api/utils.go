// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
)

// ErrorRes represents the HTTP error response body.
type ErrorRes struct {
	Err string `json:"error"`
}

var ErrInvalidQueryParams = errors.New("invalid query params")

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger *slog.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		switch err {
		case ErrLimitSize, ErrOffsetSize:
			logger.Error(err.Error())
		}
		enc(ctx, err, w)
	}
}

// ReadUintQuery reads the value of uint64 http query parameters for a given key.
func ReadUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}
	if len(vals) == 0 {
		return def, nil
	}
	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, ErrInvalidQueryParams
	}
	return val, nil
}

// ReadStringQuery reads the value of string http query parameters for a given key.
func ReadStringQuery(r *http.Request, key string, def string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", ErrInvalidQueryParams
	}
	if len(vals) == 0 {
		return def, nil
	}
	return vals[0], nil
}
