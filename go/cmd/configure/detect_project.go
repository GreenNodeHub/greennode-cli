package configure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
)

// vserverEndpointForRegion returns the vServer base URL for a region,
// looking it up in the REGIONS map.
func vserverEndpointForRegion(region string) (string, error) {
	r, ok := config.REGIONS[region]
	if !ok {
		return "", fmt.Errorf("unknown region: %s", region)
	}
	ep, ok := r["vserver_endpoint"]
	if !ok {
		return "", fmt.Errorf("no vserver_endpoint configured for region %s", region)
	}
	return ep, nil
}

// detectProjectTimeout is short — configure should fail fast, not hang the wizard.
const detectProjectTimeout = 10 * time.Second

type projectsResponse struct {
	Projects []struct {
		ProjectID string `json:"projectId"`
	} `json:"projects"`
}

// detectProjectID fetches the caller's project from vServer /v1/projects
// using the given credentials and region's vServer endpoint.
//
// Returns the first projectId. Each user is expected to have exactly one
// project per region; returning the first is safe by that contract.
func detectProjectID(clientID, clientSecret, vserverEndpoint string) (string, error) {
	// detect_project runs during `grn configure` before a profile/iam_env exists,
	// so default to prod v2 (same IAM backend as the former v1 path). The user's
	// real env is selected later per-profile via iam_env.
	tokenURL, err := login.TokenURLForEnv(login.DefaultIamEnv)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	tp := auth.NewMachineTokenProvider(clientID, clientSecret, tokenURL)
	token, err := tp.GetToken()
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	req, err := http.NewRequest("GET", vserverEndpoint+"/v1/projects", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{Timeout: detectProjectTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed projectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(parsed.Projects) == 0 {
		return "", fmt.Errorf("account has no project in this region")
	}

	return parsed.Projects[0].ProjectID, nil
}
