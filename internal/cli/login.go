package cli

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/auth"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Pipe Hub",
	Args:  noArgs("pipe login"),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug("checking existing credentials")
		existing, err := auth.LoadCredentials()
		if err != nil {
			return fmt.Errorf("reading credentials: %w", err)
		}
		if existing != nil {
			log.Debug("existing credentials found", "username", existing.Username)
			log.Warn("already logged in", "username", existing.Username)
			fmt.Print("Re-authenticate? [y/N] ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if answer := scanner.Text(); answer != "y" && answer != "Y" {
				return nil
			}
		}

		log.Debug("initiating device auth", "apiURL", apiURL)
		client := auth.NewClient(apiURL)
		info := auth.CollectDeviceInfo()
		log.Debug("device info collected", "clientName", info.ClientName, "os", info.ClientOS, "arch", info.ClientArch)

		resp, err := client.InitiateDeviceAuth(&auth.DeviceAuthRequest{
			ClientName:     info.ClientName,
			ClientOS:       info.ClientOS,
			ClientArch:     info.ClientArch,
			ClientHostname: info.ClientHostname,
		})
		if err != nil {
			return fmt.Errorf("could not reach Pipe Hub at %s: %w", apiURL, err)
		}
		log.Debug("device auth initiated", "userCode", resp.UserCode, "expiresIn", resp.ExpiresIn, "interval", resp.Interval)

		fmt.Println()
		fmt.Println("Attempting to open your default browser.")
		fmt.Println("If the browser does not open or you wish to use a different device to authorize this request, open the following URL:")
		fmt.Printf("\n  %s\n", resp.VerificationURIComplete)
		fmt.Printf("\nThen enter the code:\n\n  %s\n\n", resp.UserCode)

		if err := browser.OpenURL(resp.VerificationURIComplete); err != nil {
			log.Warn("could not open browser")
		}

		log.Debug("polling for authorization", "interval", resp.Interval, "expiresIn", resp.ExpiresIn)
		fmt.Println("Waiting for authorization...")

		status, err := auth.PollForAuthorization(client, resp.DeviceCode, resp.Interval, resp.ExpiresIn)
		if err != nil {
			return err
		}

		username := ""
		if status.Username != nil {
			username = *status.Username
		}
		apiKey := ""
		if status.APIKey != nil {
			apiKey = *status.APIKey
		}

		creds := &auth.Credentials{
			APIKey:       apiKey,
			Username:     username,
			APIBaseURL:   apiURL,
			AuthorizedAt: time.Now(),
		}
		log.Debug("saving credentials", "username", username)
		if err := auth.SaveCredentials(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
		log.Debug("credentials saved successfully")
		fmt.Printf("Successfully logged in as %s\n", username)
		return nil
	},
}
