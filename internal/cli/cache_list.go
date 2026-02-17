package cli

import (
	"fmt"

	"github.com/idestis/pipe/internal/cache"
	"github.com/spf13/cobra"
)

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached step results",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := cache.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("no cached entries")
			return nil
		}

		// Find max widths for alignment
		maxStep := len("STEP")
		for _, e := range entries {
			if len(e.StepID) > maxStep {
				maxStep = len(e.StepID)
			}
		}

		fmt.Printf("%-*s  %-20s  %-20s  %s\n", maxStep, "STEP", "CACHED AT", "EXPIRES AT", "TYPE")
		for _, e := range entries {
			cachedAt := e.CachedAt.Local().Format("2006-01-02 15:04:05")
			expiresAt := "never"
			if e.ExpiresAt != nil {
				expiresAt = e.ExpiresAt.Local().Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-*s  %-20s  %-20s  %s\n", maxStep, e.StepID, cachedAt, expiresAt, e.RunType)
		}
		return nil
	},
}
