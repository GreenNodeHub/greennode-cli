package iam

type excludedOperation struct {
	Method string
	Path   string
	Reason string
}

const reasonCredential = "returns live credential material: "

func excludedOperations() []excludedOperation {
	return []excludedOperation{
		{"POST", "/v1/auth/token", reasonCredential + "exchanges credentials for an IAM access token"},
		{"POST", "/v1/auth/service-accounts/{id}/impersonate", reasonCredential + "issues an access token for an impersonated service account"},
	}
}

func excludedIAMKeys() map[string]string {
	keys := make(map[string]string, len(excludedOperations()))
	for _, excluded := range excludedOperations() {
		keys[excluded.Method+" "+excluded.Path] = excluded.Reason
	}
	return keys
}
