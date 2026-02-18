package auth

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

type DeviceInfo struct {
	ClientName     string
	ClientOS       string
	ClientArch     string
	ClientHostname string
}

func CollectDeviceInfo() DeviceInfo {
	hostname, _ := os.Hostname()
	return DeviceInfo{
		ClientName:     "pipe-cli",
		ClientOS:       runtime.GOOS,
		ClientArch:     runtime.GOARCH,
		ClientHostname: hostname,
	}
}

func PollForAuthorization(client *Client, deviceCode string, interval, expiresIn int) (*DeviceAuthStatusResponse, error) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	deadline := time.After(time.Duration(expiresIn) * time.Second)

	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("device authorization expired")
		case <-ticker.C:
			status, err := client.PollDeviceAuthStatus(deviceCode)
			if err != nil {
				return nil, err
			}
			switch status.Status {
			case "authorized":
				return status, nil
			case "denied":
				return nil, fmt.Errorf("device authorization was denied")
			case "expired":
				return nil, fmt.Errorf("device authorization expired")
			case "pending", "verified":
				continue
			default:
				return nil, fmt.Errorf("unexpected status: %s", status.Status)
			}
		}
	}
}
