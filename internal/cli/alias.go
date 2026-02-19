package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/idestis/pipe/internal/config"
	"github.com/idestis/pipe/internal/resolve"
	"github.com/spf13/cobra"
)

var reservedNames = map[string]bool{
	"init": true, "list": true, "validate": true, "cache": true,
	"login": true, "logout": true, "pull": true, "push": true,
	"mv": true, "alias": true, "inspect": true, "switch": true,
}

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage pipeline aliases",
	Args:  noArgs("pipe alias"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return aliasListRun()
	},
}

var aliasAddCmd = &cobra.Command{
	Use:   "add <alias> <target>",
	Short: "Create an alias for a pipeline",
	Args:  exactArgs(2, "pipe alias add <alias> <pipeline>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		raw := args[0]
		target := args[1]

		owner, aName, _ := resolve.ParsePipeArg(raw)

		if !validName(aName) {
			return fmt.Errorf("invalid alias name %q — use only letters, digits, hyphens, and underscores", aName)
		}
		if owner != "" && !validOwner(owner) {
			return fmt.Errorf("invalid owner name %q — must be 4-30 characters, using only lowercase letters, digits, hyphens, and dots", owner)
		}
		if reservedNames[aName] {
			return fmt.Errorf("%q is a reserved command name", aName)
		}

		localPath := filepath.Join(config.FilesDir, aName+".yaml")
		if _, err := os.Stat(localPath); err == nil {
			return fmt.Errorf("alias %q would shadow local pipeline %q — remove the local pipeline first or choose a different alias", aName, localPath)
		}

		if err := resolve.SetAlias(aName, target); err != nil {
			return err
		}
		log.Info("created alias", "alias", aName, "target", target)
		return nil
	},
}

var aliasRmCmd = &cobra.Command{
	Use:   "rm <alias>",
	Short: "Remove an alias",
	Args:  exactArgs(1, "pipe alias rm <alias>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := resolve.DeleteAlias(name); err != nil {
			return err
		}
		log.Info("removed alias", "alias", name)
		return nil
	},
}

var aliasMvCmd = &cobra.Command{
	Use:   "mv <alias> <new-target>",
	Short: "Reassign an alias to a different target",
	Args:  exactArgs(2, "pipe alias mv <old> <new>"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		newTarget := args[1]

		if err := resolve.ReassignAlias(name, newTarget); err != nil {
			return err
		}
		log.Info("reassigned alias", "alias", name, "target", newTarget)
		return nil
	},
}

var aliasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all aliases",
	Args:  noArgs("pipe alias list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return aliasListRun()
	},
}

func aliasListRun() error {
	aliases, err := resolve.ListAliases()
	if err != nil {
		return err
	}
	if len(aliases) == 0 {
		fmt.Println("no aliases defined — use \"pipe alias add <alias> <target>\" to create one")
		return nil
	}

	maxName := len("ALIAS")
	for _, a := range aliases {
		if len(a.Name) > maxName {
			maxName = len(a.Name)
		}
	}

	fmt.Printf("%-*s  %s\n", maxName, "ALIAS", "TARGET")
	for _, a := range aliases {
		fmt.Printf("%-*s  %s\n", maxName, a.Name, a.Target)
	}
	return nil
}

func init() {
	aliasCmd.AddCommand(aliasAddCmd)
	aliasCmd.AddCommand(aliasRmCmd)
	aliasCmd.AddCommand(aliasMvCmd)
	aliasCmd.AddCommand(aliasListCmd)
}
