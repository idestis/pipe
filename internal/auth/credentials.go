package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/getpipe-dev/pipe/internal/config"
)

type Credentials struct {
	APIKey       string    `json:"api_key"`
	Username     string    `json:"username,omitempty"`
	APIBaseURL   string    `json:"api_base_url"`
	AuthorizedAt time.Time `json:"authorized_at"`
}

func SaveCredentials(creds *Credentials) error {
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config.CredentialsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(config.CredentialsPath, data, 0o600)
}

func LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(config.CredentialsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func DeleteCredentials() error {
	err := os.Remove(config.CredentialsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
