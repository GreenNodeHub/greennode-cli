package loadbalancer

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone an existing load balancer",
	Long: "Create a new load balancer from an existing one. Run " +
		"'grn vlb loadbalancer get-clone-metadata' first to see which listener " +
		"certificates the clone needs.",
	RunE: runClone,
}

func init() {
	addLBFlag(cloneCmd)
	f := cloneCmd.Flags()
	f.String("name", "", "Name of the new load balancer (required)")
	f.String("package-id", "", "Package ID for the new load balancer (required)")
	f.StringArray("certificate", nil,
		"Per-listener certificate override, shorthand "+
			"(listenerId=...,defaultCertificateId=...,sniCertificateIds=id1|id2) or JSON (repeatable)")
	f.Bool("dry-run", false, "Print the request body without creating the clone")
	for _, name := range []string{"name", "package-id"} {
		if err := cloneCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// buildCloneBody assembles the CloneLoadBalancerRequest. In shorthand form
// sniCertificateIds is a pipe-separated list, since commas already separate the
// struct's own fields.
func buildCloneBody(name, packageID string, certificates []string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name":      name,
		"packageId": packageID,
	}

	certs, err := cli.ParseStructFlags("certificate", certificates, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range certs {
		if sni, ok := c["sniCertificateIds"].(string); ok {
			c["sniCertificateIds"] = splitPipe(sni)
		}
	}
	if certs != nil {
		body["certificates"] = certs
	}

	return body, nil
}

func splitPipe(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, "|") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runClone(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	name, _ := f.GetString("name")
	packageID, _ := f.GetString("package-id")
	certificates, _ := f.GetStringArray("certificate")
	dryRun, _ := f.GetBool("dry-run")

	body, err := buildCloneBody(name, packageID, certificates)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("clone", "load balancer "+lbID+" into "+name, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.LoadBalancerSub(projectID, lbID, "clone"), body)
	if err != nil {
		return fmt.Errorf("failed to clone load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}
