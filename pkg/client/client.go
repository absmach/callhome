package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
)

const (
	HomeUrl      = "localhost:9022"
	stopWaitTime = 5 * time.Second
)

var ipEndpoints = []string{
	"https://checkip.amazonaws.com/",
	"https://ipinfo.io/ip",
	"https://api.ipify.org/",
}

func New(svc, version string, homingLogger mflog.Logger, cancel context.CancelFunc) *homingService {
	return &homingService{
		serviceName: svc,
		version:     version,
		logger:      homingLogger,
		cancel:      cancel,
		httpClient:  *http.DefaultClient,
	}
}

type homingService struct {
	serviceName string
	version     string
	logger      mflog.Logger
	cancel      context.CancelFunc
	httpClient  http.Client
}

func (hs *homingService) CallHome(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			hs.Stop()
		default:
			var data telemetryData
			var err error
			data.Service = hs.serviceName
			data.Version = hs.version
			data.LastSeen = time.Now()
			for _, endpoint := range ipEndpoints {
				ip, err := hs.getIP(endpoint)
				if err != nil {
					hs.logger.Warn(fmt.Sprintf("failed to get ip address with error: %v", err))
					continue
				}
				parsedIP, err := netip.ParseAddr(ip)
				if err != nil {
					hs.logger.Warn(fmt.Sprintf("failed to parse ip address with error: %v", err))
					continue
				}
				data.IpAddress = parsedIP.String()
				break
			}
			if err = hs.send(&data); err != nil && data.IpAddress != "" {
				hs.logger.Warn(fmt.Sprintf("failed to send telemetry data with error: %v", err))
				continue
			}
		}
		time.Sleep(time.Hour * 2)
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
	Service   string    `json:"service"`
	IpAddress string    `json:"ip_address"`
	Version   string    `json:"mainflux_version"`
	LastSeen  time.Time `json:"last_seen"`
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
	res, err := hs.httpClient.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		return fmt.Errorf("unsuccessful sending telemetry data")
	}
	return nil
}