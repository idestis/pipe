package cli

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var version = "dev"

var resumeFlag string
var apiURL string

var rootCmd = &cobra.Command{
	Use:   "pipe <pipeline> [-- KEY=value ...]",
	Short: "A lightweight pipeline runner",
	Long:  "pipe runs local automation pipelines defined in YAML.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		overrides, err := parseVarOverrides(args[1:])
		if err != nil {
			return err
		}
		return runPipeline(args[0], overrides)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	log.SetReportTimestamp(true)
	log.SetTimeFormat("15:04:05 01/02/2006")
	styles := log.DefaultStyles()
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].SetString("ERROR").MaxWidth(5)
	log.SetStyles(styles)

	rootCmd.Flags().StringVar(&resumeFlag, "resume", "", "resume a previous run by ID")
	rootCmd.SetVersionTemplate("pipe-{{.Version}}\n")

	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func initConfig() {
	if v := os.Getenv("PIPEHUB_URL"); v != "" {
		apiURL = v
		return
	}
	apiURL = "https://pipehub.net"
}

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
