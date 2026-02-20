package cli

import (
	"fmt"

	"github.com/getpipe-dev/pipe/internal/parser"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short:   "List all pipelines",
	GroupID: "core",
	Args:  noArgs("pipe list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := parser.ListAllPipelines()
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			fmt.Println("no pipelines found — use 'pipe init <name>' to create one")
			return nil
		}

		maxName := len("NAME")
		maxAlias := len("ALIAS")
		maxVer := len("VERSION")
		for _, info := range infos {
			if len(info.Name) > maxName {
				maxName = len(info.Name)
			}
			a := info.Alias
			if a == "" {
				a = "-"
			}
			if len(a) > maxAlias {
				maxAlias = len(a)
			}
			v := info.Version
			if v == "" {
				v = "-"
			}
			if len(v) > maxVer {
				maxVer = len(v)
			}
		}

		fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxName, "NAME", maxAlias, "ALIAS", maxVer, "VERSION", "DESCRIPTION")
		for _, info := range infos {
			alias := info.Alias
			if alias == "" {
				alias = "-"
			}
			version := info.Version
			if version == "" {
				version = "-"
			}
			fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxName, info.Name, maxAlias, alias, maxVer, version, info.Description)
		}
		return nil
	},
}
