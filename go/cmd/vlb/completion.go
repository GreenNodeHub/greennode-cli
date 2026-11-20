package vlb

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

func flagCompleters() map[string]cli.CompFunc {
	return map[string]cli.CompFunc{
		// VLB resources
		"load-balancer-id":              vlbclient.CompleteLoadBalancerIDs,
		"listener-id":                   vlbclient.CompleteListenerIDs,
		"pool-id":                       vlbclient.CompletePoolIDs,
		"default-pool-id":               vlbclient.CompletePoolIDs,
		"redirect-pool-id":              vlbclient.CompletePoolIDs,
		"policy-id":                     vlbclient.CompletePolicyIDs,
		"certificate-id":                vlbclient.CompleteCertificateIDs,
		"default-certificate-authority": vlbclient.CompleteCertificateIDs,
		"package-id":                    vlbclient.CompletePackageIDs,

		// Cross-service resources owned by vserver
		"subnet-id": cli.ResourceCompletion("vserver:subnet"),
		"vpc-ids":   cli.ResourceCompletion("vserver:network"),
		"zone-id":   cli.ResourceCompletion("vserver:zone"),

		// Enums straight from the OpenAPI spec
		"scheme":                 cli.FlagValues("Internet", "Internal"),
		"type":                   cli.FlagValues("Layer 4", "Layer 7", "CA", "TLS/SSL"),
		"protocol":               cli.FlagValues("HTTP", "HTTPS", "TCP", "UDP", "PROXY"),
		"algorithm":              cli.FlagValues("ROUND_ROBIN", "LEAST_CONNECTIONS", "SOURCE_IP"),
		"action":                 cli.FlagValues("REDIRECT_TO_POOL", "REDIRECT_TO_URL"),
		"default-action":         cli.FlagValues("accept", "drop"),
		"tls-security-policy":    cli.FlagValues("COMPATIBLE", "MODERN", "STRICT"),
		"alpn-protocols":         cli.FlagValues("http/1.1", "http/1.0", "h2"),
		"certificate-file":       fileExt("pem", "crt", "cer"),
		"certificate-chain-file": fileExt("pem", "crt", "cer"),
		"private-key-file":       fileExt("pem", "key"),
	}
}

// fileExt restricts file completion to the given extensions.
func fileExt(exts ...string) cli.CompFunc {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return exts, cobra.ShellCompDirectiveFilterFileExt
	}
}

// registerCompletions walks the assembled VLB tree and wires a value completer
// onto every flag whose name appears in flagCompleters. Called from vlb.go's
// init() after all AddCommand calls; cobra ignores duplicate registrations.
func registerCompletions() {
	completers := flagCompleters()
	walkCommands(VlbCmd, func(c *cobra.Command) {
		for name, fn := range completers {
			if c.Flags().Lookup(name) != nil {
				_ = c.RegisterFlagCompletionFunc(name, fn)
			}
		}
	})
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	for _, c := range cmd.Commands() {
		fn(c)
		walkCommands(c, fn)
	}
}
