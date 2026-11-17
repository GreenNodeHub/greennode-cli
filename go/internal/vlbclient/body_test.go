package vlbclient

import (
	"strings"
	"testing"
)

func TestUnwrapData(t *testing.T) {
	wrapped := map[string]interface{}{"data": map[string]interface{}{"uuid": "lb-1"}}
	if got := UnwrapData(wrapped); got["uuid"] != "lb-1" {
		t.Errorf("UnwrapData(envelope) = %v", got)
	}
	bare := map[string]interface{}{"uuid": "lb-1"}
	if got := UnwrapData(bare); got["uuid"] != "lb-1" {
		t.Errorf("UnwrapData(bare) = %v", got)
	}
	if got := UnwrapData("not an object"); len(got) != 0 {
		t.Errorf("UnwrapData(non-object) = %v, want an empty map", got)
	}
}

func TestValidateMembers(t *testing.T) {
	ok := map[string]interface{}{"ipAddress": "10.0.0.4", "port": 80, "monitorPort": 80}

	tests := []struct {
		name    string
		members []map[string]interface{}
		wantErr string
	}{
		{name: "no members is fine — a pool may start empty"},
		{name: "complete member", members: []map[string]interface{}{ok}},
		{
			name:    "missing monitorPort (the API 400s with a confusing message)",
			members: []map[string]interface{}{{"ipAddress": "10.0.0.4", "port": 80}},
			wantErr: "monitorPort is required",
		},
		{
			name:    "the error names the offending member",
			members: []map[string]interface{}{ok, {"ipAddress": "10.0.0.5", "port": 80}},
			wantErr: "--member[1]",
		},
		{
			name:    "missing ipAddress",
			members: []map[string]interface{}{{"port": 80, "monitorPort": 80}},
			wantErr: "ipAddress is required",
		},
		{
			name:    "missing port",
			members: []map[string]interface{}{{"ipAddress": "10.0.0.4", "monitorPort": 80}},
			wantErr: "port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMembers(tt.members)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}
