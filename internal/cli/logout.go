package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/getpipe-dev/pipe/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short:   "Log out of Pipe Hub",
	GroupID: "hub",
	Args:  noArgs("pipe logout"),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.LoadCredentials()
		if err != nil {
			return fmt.Errorf("reading credentials: %w", err)
		}
		if creds == nil {
			log.Info("not logged in")
			return nil
		}

		if err := auth.DeleteCredentials(); err != nil {
			return fmt.Errorf("removing credentials: %w", err)
		}

		log.Info("logged out successfully")
		return nil
	},
}
