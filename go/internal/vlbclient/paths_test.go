package vlbclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	proj     = "proj-1"
	lbID     = "lb-1"
	listener = "li-1"
	poolID   = "pool-1"
	policyID = "pol-1"
	caID     = "ca-1"
)

// builtPaths maps every URL this package can produce to the OpenAPI path
// template it implements. The templates are copied verbatim from
// openapi-vlb.json, and TestPathsCoverTheSpec below checks that this map
// accounts for every operation in the spec — so an endpoint cannot be
// implemented against a mistyped URL, nor silently left out.
func builtPaths() map[string]string {
	return map[string]string{
		LoadBalancers(proj):                          "/v2/{projectId}/loadBalancers",
		LoadBalancer(proj, lbID):                     "/v2/{projectId}/loadBalancers/{loadBalancerId}",
		LoadBalancerSub(proj, lbID, "resize"):        "/v2/{projectId}/loadBalancers/{loadBalancerId}/resize",
		LoadBalancerSub(proj, lbID, "rebalancing"):   "/v2/{projectId}/loadBalancers/{loadBalancerId}/rebalancing",
		LoadBalancerSub(proj, lbID, "bandwidth"):     "/v2/{projectId}/loadBalancers/{loadBalancerId}/bandwidth",
		LoadBalancerSub(proj, lbID, "clone"):         "/v2/{projectId}/loadBalancers/{loadBalancerId}/clone",
		LoadBalancerSub(proj, lbID, "cloneMetadata"): "/v2/{projectId}/loadBalancers/{loadBalancerId}/cloneMetadata",
		LoadBalancerSub(proj, lbID, "scale-history"): "/v2/{projectId}/loadBalancers/{loadBalancerId}/scale-history",
		Packages(proj):                               "/v2/{projectId}/loadBalancers/packages",
		AllPackages(proj):                            "/v2/{projectId}/loadBalancers/all_packages",
		Headers(proj):                                "/v2/{projectId}/loadBalancers/headers",
		LoadBalancersBySubnet(proj, "sub-1"):         "/v2/{projectId}/loadBalancers/subnet/{subnetId}",
		Listeners(proj, lbID):                        "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners",
		Listener(proj, lbID, listener):               "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}",
		Policies(proj, lbID, listener):               "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies",
		Policy(proj, lbID, listener, policyID):       "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies/{policyId}",
		Pools(proj, lbID):                            "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools",
		Pool(proj, lbID, poolID):                     "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}",
		Members(proj, lbID, poolID):                  "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}/members",
		HealthMonitor(proj, lbID, poolID):            "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}/healthMonitor",
		Certificates(proj):                           "/v2/{projectId}/cas",
		Certificate(proj, caID):                      "/v2/{projectId}/cas/{caId}",
	}
}

// TestBuiltPathsMatchTheirTemplates checks each generated URL against its
// OpenAPI template with the placeholders filled in.
func TestBuiltPathsMatchTheirTemplates(t *testing.T) {
	fill := strings.NewReplacer(
		"{projectId}", proj,
		"{loadBalancerId}", lbID,
		"{listenerId}", listener,
		"{poolId}", poolID,
		"{policyId}", policyID,
		"{l7PolicyId}", policyID,
		"{caId}", caID,
		"{subnetId}", "sub-1",
	)
	for got, template := range builtPaths() {
		if want := fill.Replace(template); got != want {
			t.Errorf("path for %s\n  got  %s\n  want %s", template, got, want)
		}
	}
}

// specPath locates openapi-vlb.json at the repo root.
func specPath() string {
	return filepath.Join("..", "..", "..", "openapi-vlb.json")
}

type openAPI struct {
	Paths map[string]map[string]struct {
		OperationID string `json:"operationId"`
	} `json:"paths"`
}

// unimplementedSpecPaths are endpoints the CLI deliberately does not expose.
// Listing them here keeps TestPathsCoverTheSpec meaningful: a genuinely
// forgotten endpoint still fails the build, while a considered omission is
// recorded with its reason.
var unimplementedSpecPaths = map[string]string{
	"/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/reorderL7Policies": "" +
		"policy reordering is not exposed: the only live attempt returned HTTP 200, " +
		"changed no position, and left the load balancer stuck in UPDATING for ~30 " +
		"minutes, refusing further changes and deletion alike. Re-add the command " +
		"once the endpoint is understood.",
}

// TestPathsCoverTheSpec is the coverage proof: every path in openapi-vlb.json
// must be produced by a builder in this package unless it is listed in
// unimplementedSpecPaths, and every builder must correspond to a real path. It
// skips when the spec is not checked out, so it never breaks a build that does
// not ship the file.
func TestPathsCoverTheSpec(t *testing.T) {
	raw, err := os.ReadFile(specPath())
	if err != nil {
		t.Skipf("openapi-vlb.json not available (%v); skipping spec conformance", err)
	}
	var spec openAPI
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parsing openapi-vlb.json: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("openapi-vlb.json declares no paths")
	}

	// The spec names the same placeholder two ways on the l7policy routes.
	normalize := strings.NewReplacer("{l7PolicyId}", "{policyId}")

	implemented := map[string]bool{}
	for _, template := range builtPaths() {
		implemented[normalize.Replace(template)] = true
	}

	var missing, extra []string
	inSpec := map[string]bool{}
	for path := range spec.Paths {
		n := normalize.Replace(path)
		inSpec[n] = true
		if _, deliberate := unimplementedSpecPaths[n]; deliberate {
			continue
		}
		if !implemented[n] {
			missing = append(missing, path)
		}
	}
	// A path that gains a builder should be dropped from the exception list,
	// so the list cannot quietly outlive its reason.
	for path := range unimplementedSpecPaths {
		if implemented[path] {
			t.Errorf("%s is implemented; remove it from unimplementedSpecPaths", path)
		}
		if !inSpec[path] {
			t.Errorf("%s is in unimplementedSpecPaths but not in the spec", path)
		}
	}
	for template := range implemented {
		if !inSpec[template] {
			extra = append(extra, template)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	for _, p := range missing {
		t.Errorf("spec path %s is not implemented by any path builder", p)
	}
	for _, p := range extra {
		t.Errorf("path builder produces %s, which is not in the spec", p)
	}
	t.Logf("%d spec paths: %d covered, %d deliberately unimplemented",
		len(spec.Paths), len(spec.Paths)-len(unimplementedSpecPaths), len(unimplementedSpecPaths))
}

// TestEveryPathIsProjectScoped guards the invariant that makes ProjectID
// mandatory: no VLB URL may be built without one.
func TestEveryPathIsProjectScoped(t *testing.T) {
	re := regexp.MustCompile(`^/v2/` + regexp.QuoteMeta(proj) + `/`)
	for got := range builtPaths() {
		if !re.MatchString(got) {
			t.Errorf("path %q is not scoped to the project", got)
		}
	}
}
