package vks

import (
	"net/url"
	"testing"
)

func TestGetWorkspaceWire(t *testing.T) {
	assertVKSReadWire(t, getWorkspaceCmd, nil, "/v1/workspace", url.Values{}, `{"projectId":"fixture-result","serviceAccountId":"fixture-account","status":"ACTIVE"}`, "projectId")
}
