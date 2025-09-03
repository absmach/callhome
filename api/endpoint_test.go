// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/callhome"
	"github.com/absmach/callhome/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestEndpointsRetrieve(t *testing.T) {
	svc := mocks.NewService(t)
	svc.On("Retrieve", mock.Anything, callhome.PageMetadata{Limit: 10}).Return(callhome.TelemetryPage{}, nil)
	h := MakeHandler(svc, trace.NewNoopTracerProvider(), slog.Default())
	server := httptest.NewServer(h)
	client := server.Client()
	testCases := []struct {
		test       string
		limit      int
		offset     int
		statuscode int
	}{
		{"successful req", 10, 0, http.StatusOK},
		{"large-limit-size", maxLimitSize + 1, 0, http.StatusBadRequest},
		{"negative-limit-size", -1, 0, http.StatusBadRequest},
	}

	for _, testCase := range testCases {
		t.Run(testCase.test, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/telemetry?limit=%d&offset=%d", server.URL, testCase.limit, testCase.offset), nil)
			assert.Nil(t, err)
			res, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, testCase.statuscode, res.StatusCode)
		})
	}
}

func TestEndpointSave(t *testing.T) {
	body := `{
		"service": "ty",
		"magistrala_version": "1.0",
		"ip_address": "41.90.185.50",
		"last_seen":"2023-03-27T17:40:50.356401087+03:00"
		}`
	svc := mocks.NewService(t)
	svc.On("Save", mock.Anything, mock.AnythingOfType("callhome.Telemetry")).Return(nil)
	h := MakeHandler(svc, noop.NewTracerProvider(), slog.Default())
	server := httptest.NewServer(h)
	client := server.Client()
	testCases := []struct {
		description string
		body        string
		contentType string
		statusCode  int
	}{
		{
			description: "success",
			body:        body,
			contentType: "application/json",
			statusCode:  http.StatusCreated,
		},
		{
			description: "malformed-request",
			body:        "{}",
			contentType: "application/json",
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "wrong-content-type",
			body:        "{}",
			contentType: "application/text",
			statusCode:  http.StatusUnsupportedMediaType,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/telemetry", server.URL), strings.NewReader(testCase.body))
			if testCase.contentType != "" {
				req.Header.Set("Content-Type", testCase.contentType)
			}
			assert.Nil(t, err)
			res, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, testCase.statusCode, res.StatusCode)
		})
	}
}
