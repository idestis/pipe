package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/auth"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Show the currently authenticated user",
	GroupID: "hub",
	Args:    noArgs("pipe whoami"),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.LoadCredentials()
		if err != nil {
			return fmt.Errorf("reading credentials: %w", err)
		}
		if creds == nil {
			log.Info("not logged in")
			return nil
		}

		baseURL := creds.APIBaseURL
		if baseURL == "" {
			baseURL = apiURL
		}
		client := auth.NewClient(baseURL)
		result, err := client.Validate(creds.APIKey)
		if err != nil {
			log.Warn("credentials are invalid, run \"pipe login\" to re-authenticate")
			return nil
		}

		fmt.Printf("Logged in as %s\n", result.Username)
		return nil
	},
}
