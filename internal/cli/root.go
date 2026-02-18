package cli

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var version = "dev"

var resumeFlag string
var apiURL string
var verbosity int

var rootCmd = &cobra.Command{
	Use:   "pipe <pipeline> [-- KEY=value ...]",
	Short: "A lightweight pipeline runner",
	Long:  "pipe runs local automation pipelines defined in YAML.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		name := args[0]
		rest := args[1:]

		// pipe <pipeline> help → show pipeline-specific usage
		if len(rest) == 1 && rest[0] == "help" {
			return showPipelineHelp(name)
		}

		overrides, err := parseVarOverrides(rest)
		if err != nil {
			return err
		}
		return runPipeline(name, overrides)
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

	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "increase output verbosity (-v verbose, -vv debug)")
	rootCmd.Flags().StringVar(&resumeFlag, "resume", "", "resume a previous run by ID")
	rootCmd.SetVersionTemplate("pipe-{{.Version}}\n")

	cobra.OnInitialize(initConfig, initVerbosity)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(mvCmd)
	rootCmd.AddCommand(aliasCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(rmCmd)
}

func initVerbosity() {
	if verbosity >= 2 {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug logging enabled")
	}
}

func initConfig() {
	if v := os.Getenv("PIPEHUB_URL"); v != "" {
		apiURL = v
		log.Debug("API URL from environment", "url", apiURL)
		return
	}
	apiURL = "https://hub.getpipe.dev"
	log.Debug("API URL default", "url", apiURL)
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
