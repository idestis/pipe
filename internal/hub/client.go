package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Client communicates with the Pipe Hub API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a hub API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	log.Debug("hub API request", "method", req.Method, "url", req.URL.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Debug("hub API request failed", "method", req.Method, "url", req.URL.String(), "err", err)
		return nil, err
	}
	log.Debug("hub API response", "method", req.Method, "url", req.URL.String(), "status", resp.StatusCode)
	return resp, nil
}

// GetPipe retrieves pipe metadata. Returns nil metadata and no error if 404.
func (c *Client) GetPipe(owner, name string) (*PipeMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/pipes/%s/%s", c.BaseURL, owner, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}
	var meta PipeMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	log.Debug("GetPipe result", "owner", owner, "name", name, "found", true)
	return &meta, nil
}

// CreatePipe creates a new pipe on the hub.
func (c *Client) CreatePipe(owner string, req *CreatePipeRequest) (*PipeMetadata, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/v1/pipes/%s", c.BaseURL, owner)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := c.do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}
	var meta PipeMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	log.Debug("CreatePipe result", "owner", owner, "name", meta.Name)
	return &meta, nil
}

// GetTag retrieves metadata for a specific tag.
func (c *Client) GetTag(owner, name, tag string) (*TagDetail, error) {
	url := fmt.Sprintf("%s/api/v1/pipes/%s/%s/tags/%s", c.BaseURL, owner, name, tag)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}
	var detail TagDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	// Normalize: API returns hash in "digest" field (sha256:<hex>) but consumers
	// expect the bare hex in SHA256. Extract it when SHA256 is not set directly.
	if detail.SHA256 == "" && strings.HasPrefix(detail.Digest, "sha256:") {
		detail.SHA256 = strings.TrimPrefix(detail.Digest, "sha256:")
	}
	sha := detail.SHA256
	if len(sha) > 12 {
		sha = sha[:12]
	}
	log.Debug("GetTag result", "tag", tag, "sha256", sha, "size", detail.SizeBytes)
	return &detail, nil
}

// DownloadTag downloads the YAML content for a tag.
func (c *Client) DownloadTag(owner, name, tag string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/pipes/%s/%s/tags/%s/download", c.BaseURL, owner, name, tag)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Debug("DownloadTag result", "tag", tag, "size", len(data))
	return data, nil
}

// DownloadByDigest downloads the YAML content by content digest.
func (c *Client) DownloadByDigest(owner, name, digest string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/pipes/%s/%s/digests/%s/download", c.BaseURL, owner, name, digest)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Debug("DownloadByDigest result", "digest", digest, "size", len(data))
	return data, nil
}

// Push pushes YAML content and assigns the given tags.
func (c *Client) Push(owner, name string, content []byte, tags []string) (*PushResponse, error) {
	url := fmt.Sprintf("%s/api/v1/pipes/%s/%s/push", c.BaseURL, owner, name)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-yaml")
	if len(tags) > 0 {
		req.Header.Set("X-Pipe-Tags", strings.Join(tags, ","))
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}
	var result PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	log.Debug("Push result", "digest", result.Digest, "tags", result.Tags, "created", result.Created, "size", result.SizeBytes)
	return &result, nil
}

func readError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("server returned %d: %s", resp.StatusCode, bytes.TrimSpace(body))
}
