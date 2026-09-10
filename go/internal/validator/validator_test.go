package validator

import "testing"

func TestValidateIDAcceptsSafeIdentifiers(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"k8s-12345", "net-A1", "pro-abc123", "hcm_03"} {
		if err := ValidateID(value, "resource ID"); err != nil {
			t.Fatalf("ValidateID(%q) returned %v", value, err)
		}
	}
}

func TestValidateIDRejectsURLInjectionCharacters(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "-k8s-123", "_k8s-123", "k8s-123-", "k8s/123", "k8s?admin=true", "k8s 123", "k8s_"} {
		if err := ValidateID(value, "cluster ID"); err == nil {
			t.Errorf("ValidateID(%q) succeeded, want validation error", value)
		}
	}
}
