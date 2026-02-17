package cli

import (
	"fmt"

	"github.com/idestis/pipe/internal/parser"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pipelines",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := parser.ListPipelines()
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			fmt.Println("no pipelines found — use 'pipe init <name>' to create one")
			return nil
		}

		// find max name width for alignment
		maxName := len("NAME")
		for _, info := range infos {
			if len(info.Name) > maxName {
				maxName = len(info.Name)
			}
		}

		fmt.Printf("%-*s  %s\n", maxName, "NAME", "DESCRIPTION")
		for _, info := range infos {
			fmt.Printf("%-*s  %s\n", maxName, info.Name, info.Description)
		}
		return nil
	},
}
