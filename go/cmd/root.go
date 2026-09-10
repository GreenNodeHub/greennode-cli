package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/greennodehub/greennode-cli/cmd/configure"
	"github.com/greennodehub/greennode-cli/cmd/login"
	internalAuth "github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	internalLogin "github.com/greennodehub/greennode-cli/internal/login"
	"github.com/spf13/cobra"
)

const cliVersion = "1.13.0" // x-release-please-version

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
	NonInteractive    bool
)

var rootCmd = &cobra.Command{
	Use:     "grn",
	Short:   "GreenNode CLI - unified command-line tool for GreenNode services",
	Version: fmt.Sprintf("%s Go/%s %s/%s", cliVersion, runtime.Version()[2:], runtime.GOOS, runtime.GOARCH),
	// Execute prints one error without usage.
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `GreenNode CLI (grn) is a unified command-line tool for managing
GreenNode services including VKS (GreenNode Kubernetes Service).

To get started, run:
  grn configure

For help on any command:
  grn <command> --help`,
	// Reject invalid global flags before execution.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if NonInteractive && (cmd == configure.ConfigureCmd || cmd == login.LoginCmd) {
			return fmt.Errorf("%s requires interaction; omit --non-interactive (use 'configure set' for scripted configuration)", cmd.CommandPath())
		}
		if err := validateGlobalFlags(cmd); err != nil {
			return usageError{err}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(func() { cli.SetNonInteractive(NonInteractive) })
	rootCmd.PersistentFlags().BoolVar(&NonInteractive, "non-interactive", false, "Never prompt; confirmations require --force")
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
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageError{err}
	})
	userAgent := "grn-cli/" + cliVersion
	client.UserAgent = userAgent
	internalAuth.UserAgent = userAgent
	internalLogin.UserAgent = userAgent

	rootCmd.AddCommand(configure.ConfigureCmd)
	rootCmd.AddCommand(login.LoginCmd)
	rootCmd.AddCommand(login.LogoutCmd)
	for _, svc := range cli.Services() {
		rootCmd.AddCommand(svc)
	}
}

// Execute runs the root command.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitCode(err))
	}
}

type usageError struct {
	err error
}

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

type exitCoder interface {
	error
	ExitCode() int
}

func exitCode(err error) int {
	var coded exitCoder
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	var target usageError
	if errors.As(err, &target) {
		return 2
	}
	return 1
}

// ExecuteContext propagates cancellation and confirmation failures.
func ExecuteContext(ctx context.Context) error {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return cli.ConfirmationError()
}
