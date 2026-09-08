package vlb

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// These tests drive the real cobra tree the way a user does — parsing actual
// argv — and assert on what --dry-run prints. That covers the glue between flag
// parsing and the pure body builders, which the builders_test.go files cannot
// reach on their own. Only commands whose --dry-run returns before any HTTP
// call are covered here; the read-modify-write updates are exercised by their
// builders' tests with fixture state.

// resetFlags returns a command's flags to their declared defaults so table
// entries cannot leak state into each other through the shared command tree.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// runVlb executes `grn vlb <args...>` in-process and captures everything the
// command writes to stdout (the dry-run printers use fmt.Print* directly).
func runVlb(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags(VlbCmd)

	silenceErrors, silenceUsage := VlbCmd.SilenceErrors, VlbCmd.SilenceUsage
	VlbCmd.SilenceErrors, VlbCmd.SilenceUsage = true, true
	defer func() { VlbCmd.SilenceErrors, VlbCmd.SilenceUsage = silenceErrors, silenceUsage }()

	real := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	VlbCmd.SetArgs(args)
	VlbCmd.SetOut(io.Discard)
	VlbCmd.SetErr(io.Discard)
	runErr := VlbCmd.Execute()

	_ = w.Close()
	os.Stdout = real
	return <-done, runErr
}

func TestDryRunRequestBodies(t *testing.T) {
	tests := []struct {
		name string
		args []string
		// want are substrings that must appear in the dry-run output.
		want []string
		// absent are substrings that must NOT appear (fields the API would
		// reject, or optional fields that should be omitted when unset).
		absent []string
	}{
		{
			name: "loadbalancer create sends only the required fields",
			args: []string{"loadbalancer", "create", "--name", "my-lb", "--package-id", "pkg-1",
				"--scheme", "Internet", "--subnet-id", "sub-1", "--type", "Layer 7", "--dry-run"},
			want:   []string{"name: my-lb", "packageId: pkg-1", "scheme: Internet", "subnetId: sub-1", "type: Layer 7"},
			absent: []string{"zoneId", "autoScalable", "tags", "listener", "pool"},
		},
		{
			// --listener and --pool must be given together (@LbValid), and the
			// inline pool needs a nested healthMonitor, which the k=v shorthand
			// cannot express — hence JSON for the pool.
			name: "loadbalancer create with a nested listener coerces the port to an int",
			args: []string{"loadbalancer", "create", "--name", "my-lb", "--package-id", "pkg-1",
				"--scheme", "Internal", "--subnet-id", "sub-1", "--type", "Layer 4",
				"--zone-id", "HCM03-1A", "--auto-scalable",
				"--tag", "key=env,value=prod",
				"--listener", "listenerName=web,listenerProtocol=HTTP,listenerProtocolPort=80",
				"--pool", `{"poolName":"web","poolProtocol":"HTTP","algorithm":"ROUND_ROBIN",` +
					`"stickiness":false,"tlsEncryption":false,` +
					`"healthMonitor":{"healthCheckProtocol":"TCP","timeout":5,"interval":30,` +
					`"healthyThreshold":3,"unhealthyThreshold":3}}`,
				"--dry-run"},
			want: []string{"zoneId: HCM03-1A", "autoScalable: true",
				"key:env value:prod", "listenerProtocolPort:80",
				// The inline listener's three @NotNull timeouts are defaulted the
				// way `listener create` defaults them; without this the API
				// answered "listener.timeoutClient: must not be null".
				"timeoutClient:50", "timeoutMember:50", "timeoutConnection:5"},
		},
		{
			name: "loadbalancer resize",
			args: []string{"lb", "resize", "--load-balancer-id", "lb-1", "--package-id", "pkg-2", "--dry-run"},
			want: []string{"Would resize load balancer lb-1", "packageId: pkg-2"},
		},
		{
			name: "loadbalancer clone splits sni certificates on the pipe",
			args: []string{"loadbalancer", "clone", "--load-balancer-id", "lb-1",
				"--name", "lb-copy", "--package-id", "pkg-1",
				"--certificate", "listenerId=li-1,defaultCertificateId=ca-1,sniCertificateIds=ca-2|ca-3",
				"--dry-run"},
			want: []string{"name: lb-copy", "sniCertificateIds:[ca-2 ca-3]"},
		},
		{
			name: "loadbalancer configure-scaling omits minSize unless set",
			args: []string{"loadbalancer", "configure-scaling", "--load-balancer-id", "lb-1",
				"--max-size", "6", "--subnets", "sub-1,sub-2", "--dry-run"},
			want:   []string{"maxSize:6", "subnets:[sub-1 sub-2]"},
			absent: []string{"minSize"},
		},
		{
			name: "loadbalancer configure-scaling includes minSize when set",
			args: []string{"loadbalancer", "configure-scaling", "--load-balancer-id", "lb-1",
				"--min-size", "2", "--max-size", "6", "--subnets", "sub-1", "--dry-run"},
			want: []string{"minSize:2", "maxSize:6"},
		},
		{
			name: "loadbalancer update-bandwidth",
			args: []string{"lb", "update-bandwidth", "--load-balancer-id", "lb-1",
				"--bandwidth-package-id", "bandwidth-pkg-1", "--dry-run"},
			want: []string{"bandWidthPackageId: bandwidth-pkg-1"},
		},
		{
			name: "listener create always sends the three required timeouts",
			args: []string{"listener", "create", "--load-balancer-id", "lb-1",
				"--name", "http-80", "--protocol", "HTTP", "--protocol-port", "80", "--dry-run"},
			want: []string{"listenerName: http-80", "listenerProtocol: HTTP",
				"listenerProtocolPort: 80", "timeoutClient: 50", "timeoutMember: 50", "timeoutConnection: 5"},
			absent: []string{"defaultPoolId", "allowedCidrs", "insertHeaders", "tlsSecurityPolicy"},
		},
		{
			name: "listener create parses comma lists and repeatable headers",
			args: []string{"listener", "create", "--load-balancer-id", "lb-1",
				"--name", "https-443", "--protocol", "HTTPS", "--protocol-port", "443",
				"--certificate-authorities", "ca-1, ca-2",
				"--alpn-protocols", "h2,http/1.1",
				"--insert-header", "headerName=X-Forwarded-For,headerValue=true",
				"--tls-security-policy", "MODERN", "--dry-run"},
			want: []string{"certificateAuthorities: [ca-1 ca-2]", "alpnProtocols: [h2 http/1.1]",
				"headerName:X-Forwarded-For", "tlsSecurityPolicy: MODERN"},
		},
		{
			name: "pool create nests the health monitor and members",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1",
				"--name", "web-pool", "--protocol", "HTTP", "--algorithm", "ROUND_ROBIN",
				"--health-monitor", "healthCheckProtocol=HTTP,healthCheckPath=/healthz,healthCheckMethod=GET," +
					"successCode=200,httpVersion=1.0,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
				"--member", "ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false",
				"--member", "ipAddress=10.0.0.5,port=80,monitorPort=80",
				"--dry-run"},
			want: []string{"poolName: web-pool", "poolProtocol: HTTP", "algorithm: ROUND_ROBIN",
				"healthCheckPath:/healthz", "timeout:5", "interval:30",
				"ipAddress:10.0.0.4", "port:80", "backup:false", "ipAddress:10.0.0.5"},
		},
		{
			name: "policy create for a pool redirect omits the url",
			args: []string{"listener", "policy", "create", "--load-balancer-id", "lb-1",
				"--listener-id", "li-1", "--name", "route-api", "--action", "REDIRECT_TO_POOL",
				"--redirect-pool-id", "pool-api",
				"--rule", "ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api", "--dry-run"},
			want:   []string{"action: REDIRECT_TO_POOL", "redirectPoolId: pool-api", "ruleValue:/api"},
			absent: []string{"redirectUrl"},
		},
		{
			name: "policy create for a url redirect omits the pool",
			args: []string{"listener", "policy", "create", "--load-balancer-id", "lb-1",
				"--listener-id", "li-1", "--name", "to-https", "--action", "REDIRECT_TO_URL",
				"--redirect-url", "https://example.com", "--redirect-http-code", "301",
				"--keep-query-string",
				"--rule", "ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=old.example.com",
				"--dry-run"},
			want:   []string{"redirectUrl: https://example.com", "redirectHttpCode: 301", "keepQueryString: true"},
			absent: []string{"redirectPoolId"},
		},
		{
			name: "certificate create redacts secret material",
			args: []string{"certificate", "create", "--name", "web-cert", "--type", "TLS/SSL",
				"--certificate", "-----BEGIN CERTIFICATE-----abc",
				"--private-key", "-----BEGIN PRIVATE KEY-----supersecret",
				"--passphrase", "hunter2", "--dry-run"},
			want:   []string{"name: web-cert", "privateKey: <redacted>", "passphrase: <redacted>", "bytes of PEM"},
			absent: []string{"supersecret", "hunter2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runVlb(t, tt.args...)
			if err != nil {
				t.Fatalf("command failed: %v\noutput:\n%s", err, out)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output is missing %q\n--- got ---\n%s", want, out)
				}
			}
			for _, absent := range tt.absent {
				if strings.Contains(out, absent) {
					t.Errorf("output should not contain %q\n--- got ---\n%s", absent, out)
				}
			}
			if !strings.Contains(out, "Run without --dry-run to") {
				t.Errorf("dry-run output is missing the standard footer\n--- got ---\n%s", out)
			}
		})
	}
}

// TestDryRunRejectsBadInput checks that argument validation happens before any
// network call, so a typo fails fast with an actionable message.
func TestDryRunRejectsBadInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing required flag",
			args:    []string{"loadbalancer", "create", "--name", "my-lb", "--dry-run"},
			wantErr: "required flag",
		},
		{
			// @LbValid: "Both listener and pool properties must be required
			// (non null) or both are not required (null)". Caught locally
			// instead of as a 400 with no field name.
			name: "an inline listener without a pool is refused",
			args: []string{"loadbalancer", "create", "--name", "my-lb", "--package-id", "pkg-1",
				"--scheme", "Internet", "--subnet-id", "sub-1", "--type", "Layer 7",
				"--listener", "listenerName=web,listenerProtocol=HTTP,listenerProtocolPort=80",
				"--dry-run"},
			wantErr: "must all be set",
		},
		{
			name: "an inline pool without a listener is refused",
			args: []string{"loadbalancer", "create", "--name", "my-lb", "--package-id", "pkg-1",
				"--scheme", "Internet", "--subnet-id", "sub-1", "--type", "Layer 7",
				"--pool", "poolName=web,poolProtocol=HTTP,algorithm=ROUND_ROBIN",
				"--dry-run"},
			wantErr: "must all be set",
		},
		{
			// The API tests stickiness with != null, so a non-HTTP pool must not
			// carry it at all. Refuse rather than silently drop the user's flag.
			name: "stickiness on a non-HTTP pool is refused",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p1",
				"--protocol", "PROXY", "--stickiness",
				"--health-monitor", "healthCheckProtocol=TCP,timeout=5,interval=30," +
					"healthyThreshold=3,unhealthyThreshold=3",
				"--dry-run"},
			wantErr: "only valid for an HTTP pool",
		},
		{
			name: "invalid id characters are rejected before the request",
			args: []string{"loadbalancer", "resize", "--load-balancer-id", "lb_bad",
				"--package-id", "pkg-1", "--dry-run"},
			wantErr: "invalid load-balancer-id",
		},
		{
			name: "listener port out of range",
			args: []string{"listener", "create", "--load-balancer-id", "lb-1", "--name", "x",
				"--protocol", "HTTP", "--protocol-port", "70000", "--dry-run"},
			wantErr: "--protocol-port must be between 1 and 65535",
		},
		{
			name: "listener timeout out of range",
			args: []string{"listener", "create", "--load-balancer-id", "lb-1", "--name", "x",
				"--protocol", "HTTP", "--protocol-port", "80", "--timeout-client", "9999", "--dry-run"},
			wantErr: "--timeout-client must be between 1 and 3600",
		},
		{
			name: "non-integer value in struct shorthand",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "HTTP", "--health-monitor", "timeout=soon", "--dry-run"},
			wantErr: "timeout must be an integer",
		},
		{
			name: "struct shorthand without an equals sign",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "HTTP",
				"--health-monitor", "healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
				"--member", "10.0.0.4:80", "--dry-run"},
			wantErr: "expected key=value",
		},
		{
			name: "health monitor missing the fields the API demands",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "HTTP", "--health-monitor", "healthCheckProtocol=TCP", "--dry-run"},
			wantErr: "missing required field(s): timeout, interval, healthyThreshold, unhealthyThreshold",
		},
		{
			name: "HTTP health monitor missing its extra fields",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "HTTP",
				"--health-monitor", "healthCheckProtocol=HTTP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
				"--dry-run"},
			wantErr: "healthCheckPath, healthCheckMethod, successCode, httpVersion",
		},
		{
			name: "httpVersion 1.1 without a domain name",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "HTTP",
				"--health-monitor", "healthCheckProtocol=HTTP,healthCheckPath=/,healthCheckMethod=GET," +
					"successCode=200,httpVersion=1.1,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
				"--dry-run"},
			wantErr: "domainName is required when httpVersion is 1.1",
		},
		{
			name: "member without monitorPort",
			args: []string{"pool", "create", "--load-balancer-id", "lb-1", "--name", "p",
				"--protocol", "TCP",
				"--health-monitor", "healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
				"--member", "ipAddress=10.0.0.4,port=80", "--dry-run"},
			wantErr: "monitorPort is required by the API",
		},
		{
			name: "policy redirect target must match the action",
			args: []string{"listener", "policy", "create", "--load-balancer-id", "lb-1",
				"--listener-id", "li-1", "--name", "p", "--action", "REDIRECT_TO_POOL",
				"--rule", "ruleType=PATH,compareType=STARTS_WITH,ruleValue=/x", "--dry-run"},
			wantErr: "--redirect-pool-id is required",
		},
		{
			name: "unknown redirect http code",
			args: []string{"listener", "policy", "create", "--load-balancer-id", "lb-1",
				"--listener-id", "li-1", "--name", "p", "--action", "REDIRECT_TO_URL",
				"--redirect-url", "https://x", "--redirect-http-code", "307",
				"--rule", "ruleType=PATH,compareType=STARTS_WITH,ruleValue=/x", "--dry-run"},
			wantErr: "must be 301 or 302",
		},
		{
			name: "policy create with no rule (the API rejects a ruleless policy)",
			args: []string{"listener", "policy", "create", "--load-balancer-id", "lb-1",
				"--listener-id", "li-1", "--name", "p", "--action", "REDIRECT_TO_POOL",
				"--redirect-pool-id", "pool-1", "--dry-run"},
			wantErr: "--rule is required",
		},
		{
			name: "certificate type must be one of the two",
			args: []string{"certificate", "create", "--name", "c", "--type", "PGP",
				"--certificate", "PEM", "--dry-run"},
			wantErr: `--type must be "CA" or "TLS/SSL"`,
		},
		{
			name: "TLS/SSL certificate without a private key",
			args: []string{"certificate", "create", "--name", "c", "--type", "TLS/SSL",
				"--certificate", "PEM", "--dry-run"},
			wantErr: "a private key is required",
		},
		{
			name: "certificate inline and file flags are mutually exclusive",
			args: []string{"certificate", "create", "--name", "c", "--type", "CA",
				"--certificate", "PEM", "--certificate-file", "./x.pem", "--dry-run"},
			wantErr: "were all set",
		},
		{
			name: "scaling requires a positive max size",
			args: []string{"loadbalancer", "configure-scaling", "--load-balancer-id", "lb-1",
				"--max-size", "0", "--subnets", "sub-1", "--dry-run"},
			wantErr: "--max-size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runVlb(t, tt.args...)
			if err == nil {
				t.Fatalf("expected an error, got none\noutput:\n%s", out)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
