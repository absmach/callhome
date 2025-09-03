// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"time"
)

var (
	// ErrLimitSize indicates that an invalid limit.
	ErrLimitSize = errors.New("invalid limit size")
	// ErrOffsetSize indicates an invalid offset.
	ErrOffsetSize = errors.New("invalid offset size")
	// ErrInvalidDateRange indicates date from and to are invalid.
	ErrInvalidDateRange = errors.New("invalid date range")
	// ErrMalformedEntity represents malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity")
	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type.
	ErrUnsupportedContentType = errors.New("unsupported content type")
)

const maxLimitSize = 100

type saveTelemetryReq struct {
	Service   string    `json:"service"`
	IpAddress string    `json:"ip_address"`
	Version   string    `json:"magistrala_version"`
	LastSeen  time.Time `json:"last_seen"`
}

func (req saveTelemetryReq) validate() error {
	if req.Service == "" {
		return ErrMalformedEntity
	}

	if req.IpAddress == "" {
		return ErrMalformedEntity
	}
	if req.Version == "" {
		return ErrMalformedEntity
	}

	return nil
}

type listTelemetryReq struct {
	offset  uint64
	limit   uint64
	from    time.Time
	to      time.Time
	country string
	city    string
	version string
	service string
}

func (req listTelemetryReq) validate() error {
	if req.limit > maxLimitSize || req.limit < 1 {
		return ErrLimitSize
	}

	if !req.from.IsZero() && !req.to.IsZero() && req.to.Before(req.from) {
		return ErrInvalidDateRange
	}

	return nil
}
