package cli

import (
	"fmt"

	"github.com/getpipe-dev/pipe/internal/cache"
	"github.com/spf13/cobra"
)

var cacheClearCmd = &cobra.Command{
	Use:   "clear [step-id]",
	Short: "Clear one or all cache entries",
	Args:  maxArgs(1, "pipe cache clear [step-id]"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			stepID := args[0]
			if err := cache.Clear(stepID); err != nil {
				return err
			}
			fmt.Printf("cleared cache for %q\n", stepID)
			return nil
		}

		if err := cache.ClearAll(); err != nil {
			return err
		}
		fmt.Println("cleared all cache entries")
		return nil
	},
}
