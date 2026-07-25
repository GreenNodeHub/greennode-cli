package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/cmd/configure"
	"github.com/vngcloud/greennode-cli/internal/cli"
	"github.com/vngcloud/greennode-cli/internal/config"
)

const cliVersion = "1.7.2" // x-release-please-version

// Global flags
var (
	Profile           string
	Region            string
	Output            string
	Query             string
	EndpointURL       string
	NoVerifySSL       bool
	Debug             bool
	CLIReadTimeout    int
	CLIConnectTimeout int
	Color             string
	AllowUntrusted    bool
)

var rootCmd = &cobra.Command{
	Use:     "grn",
	Short:   "GreenNode CLI - unified command-line tool for GreenNode services",
	Version: fmt.Sprintf("%s Go/%s %s/%s", cliVersion, runtime.Version()[2:], runtime.GOOS, runtime.GOARCH),
	// Print a single clean "Error: ..." line on failure (done in Execute) rather
	// than cobra's error plus a full usage dump.
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `GreenNode CLI (grn) is a unified command-line tool for managing
GreenNode services including VKS (GreenNode Kubernetes Service).

To get started, run:
  grn configure

For help on any command:
  grn <command> --help`,
	// Validate global flags up front so an invalid --output fails fast with a
	// suggestion, rather than silently falling back to JSON.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return validateGlobalFlags(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&Profile, "profile", "", "Use a specific profile from credentials file")
	rootCmd.PersistentFlags().StringVar(&Region, "region", "", "The region to use (e.g. HCM-3, HAN)")
	rootCmd.PersistentFlags().StringVar(&Output, "output", "", "The output format (json, text, table)")
	rootCmd.PersistentFlags().StringVar(&Query, "query", "", "JMESPath query to filter output")
	rootCmd.PersistentFlags().StringVar(&EndpointURL, "endpoint-url", "", "Override the service endpoint URL")
	rootCmd.PersistentFlags().BoolVar(&NoVerifySSL, "no-verify-ssl", false, "Disable SSL certificate verification")
	rootCmd.PersistentFlags().BoolVar(&AllowUntrusted, "allow-untrusted-endpoint", false, "Allow --endpoint-url to a host outside vngcloud.vn/greennode.ai without TLS protection (sends a bearer token there)")
	rootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().IntVar(&CLIReadTimeout, "cli-read-timeout", 30, "HTTP read timeout in seconds")
	rootCmd.PersistentFlags().IntVar(&CLIConnectTimeout, "cli-connect-timeout", 30, "HTTP connect timeout in seconds")
	rootCmd.PersistentFlags().StringVar(&Color, "color", "auto", "Color output (on, off, auto)")

	_ = rootCmd.RegisterFlagCompletionFunc("region", cli.FlagValuesFrom(config.RegionNames))
	_ = rootCmd.RegisterFlagCompletionFunc("profile", cli.FlagValuesFrom(config.ProfileNames))
	_ = rootCmd.RegisterFlagCompletionFunc("output", cli.FlagValues("json", "text", "table"))
	_ = rootCmd.RegisterFlagCompletionFunc("color", cli.FlagValues("on", "off", "auto"))

	rootCmd.SetVersionTemplate("grn-cli/{{.Version}}\n")

	rootCmd.AddCommand(configure.ConfigureCmd)
	for _, svc := range cli.Services() {
		rootCmd.AddCommand(svc)
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
