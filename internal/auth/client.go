package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type DeviceAuthRequest struct {
	ClientName     string `json:"clientName"`
	ClientOS       string `json:"clientOS"`
	ClientArch     string `json:"clientArch"`
	ClientHostname string `json:"clientHostname"`
}

type DeviceAuthResponse struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

type DeviceAuthStatusResponse struct {
	Status   string  `json:"status"`
	APIKey   *string `json:"apiKey,omitempty"`
	Username *string `json:"username,omitempty"`
}

func (c *Client) InitiateDeviceAuth(req *DeviceAuthRequest) (*DeviceAuthResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/api/v1/auth/device",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var result DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

func (c *Client) PollDeviceAuthStatus(deviceCode string) (*DeviceAuthStatusResponse, error) {
	resp, err := c.HTTPClient.Get(
		c.BaseURL + "/api/v1/auth/device/status?device_code=" + deviceCode,
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var result DeviceAuthStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}
