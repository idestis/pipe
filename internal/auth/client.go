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
	defer func() { _ = resp.Body.Close() }()

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

// Logout revokes the device and API key on the server.
// Treats 204 and 401 as success (key already revoked is fine for logout).
func (c *Client) Logout(apiKey string) error {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/auth/device/logout", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusUnauthorized {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("server returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
}

type ValidateResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

// Validate checks if the API key is still valid by calling GET /api/v1/users/me.
func (c *Client) Validate(apiKey string) (*ValidateResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("credentials are invalid or revoked")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var result ValidateResponse
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
	defer func() { _ = resp.Body.Close() }()

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
