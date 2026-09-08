package vserver

import "testing"

func TestSubnetPathRequiresVPC(t *testing.T) {
	if p, ok := subnetPath("proj1", "vpc1"); !ok || p != "/v2/proj1/networks/vpc1/subnets" {
		t.Errorf("subnetPath = %q ok=%v", p, ok)
	}
	if _, ok := subnetPath("proj1", ""); ok {
		t.Errorf("subnetPath with empty vpc should be !ok")
	}
	if _, ok := subnetPath("", "vpc1"); ok {
		t.Errorf("subnetPath with empty project should be !ok")
	}
}

func TestListPath(t *testing.T) {
	if got := listPath("proj1", "/v2/%s/networks"); got != "/v2/proj1/networks" {
		t.Errorf("listPath = %q", got)
	}
}

func TestRegisteredKeys(t *testing.T) {
	keys := registeredKeys()
	for _, k := range []string{"vserver:network", "vserver:subnet", "vserver:sshkey", "vserver:secgroup", "vserver:volumetype", "vserver:zone"} {
		if keys[k] == nil {
			t.Errorf("key %q not present in registeredKeys()", k)
		}
	}
	if len(keys) != 6 {
		t.Errorf("registeredKeys() has %d entries, want 6", len(keys))
	}
}
