// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	HomeUrl           = "https://deployments.absmach.eu/telemetry"
	stopWaitTime      = 5 * time.Second
	callHomeSleepTime = 30 * time.Minute
	backOff           = 10 * time.Second
	apiKey            = "77e04a7c-f207-40dd-8950-c344871fd516"
	defDeploymentID   = "/var/lib/magistrala/callhome/deployment_id"
)

var ipEndpoints = []string{
	"https://checkip.amazonaws.com/",
	"https://ipinfo.io/ip",
	"https://api.ipify.org/",
}

type homingService struct {
	serviceName  string
	version      string
	deploymentID string
	logger       *slog.Logger
	cancel       context.CancelFunc
	httpClient   http.Client
}

func New(svc, version string, homingLogger *slog.Logger, cancel context.CancelFunc) *homingService {
	return &homingService{
		serviceName:  svc,
		version:      version,
		deploymentID: getDeploymentID(homingLogger),
		logger:       homingLogger,
		cancel:       cancel,
		httpClient:   *http.DefaultClient,
	}
}

func (hs *homingService) CallHome(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			hs.Stop()
		default:
			data := telemetryData{
				Service:      hs.serviceName,
				Version:      hs.version,
				DeploymentID: hs.deploymentID,
				LastSeen:     time.Now(),
			}

			var macAddr string
			interfaces, err := net.Interfaces()
			if err != nil {
				hs.logger.Warn(fmt.Sprintf("failed to obtain MAC address: %v", err))
				continue
			}

			for _, i := range interfaces {
				mac := i.HardwareAddr
				if len(mac) != 0 {
					macAddr = i.HardwareAddr.String()
					break
				}
			}
			data.MACAddress = macAddr
			for _, endpoint := range ipEndpoints {
				ip, err := hs.getIP(endpoint)
				if err != nil {
					hs.logger.Warn(fmt.Sprintf("failed to obtain service public IP address for sending Magistrala usage telemetry with error: %v", err))
					continue
				}
				ip = strings.ReplaceAll(ip, "\n", "")
				ip = strings.ReplaceAll(ip, "\\", "")
				parsedIP, err := netip.ParseAddr(ip)
				if err != nil {
					hs.logger.Warn(fmt.Sprintf("failed to parse ip address with error: %v", err))
					continue
				}
				data.IPAddress = parsedIP.String()
				break
			}
			if err := hs.send(&data); err != nil && data.IPAddress != "" {
				hs.logger.Warn(fmt.Sprintf("failed to send Magistrala telemetry data with error: %v", err))
				time.Sleep(backOff)
				continue
			}
		}
		time.Sleep(callHomeSleepTime)
	}
}

func (hs *homingService) Stop() {
	defer hs.cancel()
	c := make(chan bool)
	defer close(c)
	select {
	case <-c:
	case <-time.After(stopWaitTime):
	}
	hs.logger.Info("call home service shutdown")
}

type telemetryData struct {
	Service   string `json:"service"`
	IPAddress string `json:"ip_address"`
	// MAC address is used to identify unique machine to avoid duplicates in case
	// of multiple services running on the same machine (such as a Docker composition).
	MACAddress   string    `json:"mac_address"`
	DeploymentID string    `json:"deployment_id"`
	Version      string    `json:"magistrala_version"`
	LastSeen     time.Time `json:"last_seen"`
}

func (hs *homingService) getIP(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	res, err := hs.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (hs *homingService) send(telDat *telemetryData) error {
	b, err := json.Marshal(telDat)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, HomeUrl, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)
	res, err := hs.httpClient.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		if res != nil {
			return fmt.Errorf("unsuccessful sending telemetry data with code %d and error: %v", res.StatusCode, err)
		}
		return err
	}
	return nil
}

func getDeploymentID(logger *slog.Logger) string {
	if id := os.Getenv("MG_DEPLOYMENT_ID"); id != "" {
		return id
	}

	path := defDeploymentID
	if p := os.Getenv("MG_CALLHOME_DEPLOYMENT_ID_FILE"); p != "" {
		path = p
	}

	if id, err := os.ReadFile(path); err == nil {
		return string(id)
	}

	id := uuid.New().String()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Warn(fmt.Sprintf("failed to create directory for deployment id with error: %v", err))
		return ""
	}

	if err := os.WriteFile(path, []byte(id), 0o644); err != nil {
		logger.Warn(fmt.Sprintf("failed to write deployment id with error: %v", err))
		return ""
	}

	return id
}
