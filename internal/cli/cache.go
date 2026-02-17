package cli

import "github.com/spf13/cobra"

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage step cache",
}

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheClearCmd)
}
