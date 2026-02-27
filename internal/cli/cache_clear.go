package cli

import (
	"fmt"

	"github.com/getpipe-dev/pipe/internal/cache"
	"github.com/spf13/cobra"
)

var cacheClearYes bool

func init() {
	cacheClearCmd.Flags().BoolVarP(&cacheClearYes, "yes", "y", false, "skip confirmation prompt")
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear [step-id]",
	Short: "Clear one or all cache entries",
	Args:  maxArgs(1, "pipe cache clear [step-id]"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			stepID := args[0]
			if !confirmAction(cacheClearYes, fmt.Sprintf("Clear cache for %q?", stepID)) {
				return nil
			}
			if err := cache.Clear(stepID); err != nil {
				return err
			}
			fmt.Printf("cleared cache for %q\n", stepID)
			return nil
		}

		if !confirmAction(cacheClearYes, "Clear all cache entries?") {
			return nil
		}
		if err := cache.ClearAll(); err != nil {
			return err
		}
		fmt.Println("cleared all cache entries")
		return nil
	},
}
