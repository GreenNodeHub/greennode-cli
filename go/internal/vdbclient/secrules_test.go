package vdbclient

import "testing"

const testPort = 5432

func TestParseSecurityRules(t *testing.T) {
	rules, err := ParseSecurityRules([]string{
		"cidr=203.0.113.0/24",
		"cidr=10.0.0.0/8,port-min=5432,port-max=5433",
		"cidr=192.0.2.1/32,port=6432,id=rule-1",
	}, testPort)
	if err != nil {
		t.Fatalf("ParseSecurityRules: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("got %d rules, want 3", len(rules))
	}

	first := rules[0].(map[string]interface{})
	if first["portRangeMin"] != testPort || first["portRangeMax"] != testPort {
		t.Errorf("a rule without a port should default to %d, got %v-%v",
			testPort, first["portRangeMin"], first["portRangeMax"])
	}
	if _, present := first["id"]; present {
		t.Error("a new rule must not carry an id; the API reads a null id as 'insert'")
	}
	if first["remoteIpPrefix"] != "203.0.113.0/24" {
		t.Errorf("remoteIpPrefix = %v", first["remoteIpPrefix"])
	}

	second := rules[1].(map[string]interface{})
	if second["portRangeMin"] != 5432 || second["portRangeMax"] != 5433 {
		t.Errorf("port range = %v-%v, want 5432-5433", second["portRangeMin"], second["portRangeMax"])
	}

	third := rules[2].(map[string]interface{})
	if third["id"] != "rule-1" || third["portRangeMin"] != 6432 {
		t.Errorf("rule with id = %v", third)
	}
}

// TestParseSecurityRulesDefaultPortIsPerEngine: MySQL/MariaDB listen on 3306,
// PostgreSQL on 5432, Redis on 6379, so the default is the caller's to pass.
func TestParseSecurityRulesDefaultPortIsPerEngine(t *testing.T) {
	for _, port := range []int{3306, 5432, 6379} {
		rules, err := ParseSecurityRules([]string{"cidr=0.0.0.0/0"}, port)
		if err != nil {
			t.Fatalf("port %d: %v", port, err)
		}
		if got := rules[0].(map[string]interface{})["portRangeMin"]; got != port {
			t.Errorf("default port = %v, want %d", got, port)
		}
	}
}

func TestParseSecurityRulesRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"typo'd key would revoke everything": "cird=1.2.3.4/32",
		"no cidr":                            "port=5432",
		"not key=value":                      "cidr",
		"port with a range":                  "cidr=1.2.3.4/32,port=5432,port-min=5432",
		"half a range":                       "cidr=1.2.3.4/32,port-min=5432",
		"inverted range":                     "cidr=1.2.3.4/32,port-min=5433,port-max=5432",
		"port is not a number":               "cidr=1.2.3.4/32,port=abc",
		"port out of range":                  "cidr=1.2.3.4/32,port=70000",
	}

	for name, spec := range cases {
		if _, err := ParseSecurityRules([]string{spec}, testPort); err == nil {
			t.Errorf("%s: ParseSecurityRules(%q) = nil error, want a rejection", name, spec)
		}
	}

	if _, err := ParseSecurityRules(nil, testPort); err == nil {
		t.Error("no rules accepted; an empty rule set would revoke all access")
	}
}

// TestSecurityRulesFrom: re-sending an existing rule must keep its ID, otherwise
// the replace-semantics endpoint deletes and recreates it.
func TestSecurityRulesFrom(t *testing.T) {
	response := map[string]interface{}{
		"code": 200.0, "message": "ok",
		"data": []interface{}{
			map[string]interface{}{
				"id": "rule-1", "portRangeMin": 5432.0, "portRangeMax": 5432.0,
				"remoteIpPrefix": "0.0.0.0/0", "direction": "ingress", "status": "",
			},
		},
	}

	rules := SecurityRulesFrom(response)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	rule := rules[0].(map[string]interface{})
	if rule["id"] != "rule-1" {
		t.Errorf("id = %v, want it preserved", rule["id"])
	}
	// Only the four update fields survive; direction/status are read-only.
	if _, present := rule["direction"]; present {
		t.Error("direction must not be sent back on an update")
	}

	if got := SecurityRulesFrom("not a list"); got != nil {
		t.Errorf("SecurityRulesFrom(non-list) = %v, want nil", got)
	}
}

func TestDescribeSecurityRules(t *testing.T) {
	rules := []interface{}{
		map[string]interface{}{"remoteIpPrefix": "0.0.0.0/0", "portRangeMin": 5432, "portRangeMax": 5432},
		map[string]interface{}{"remoteIpPrefix": "10.0.0.0/8", "portRangeMin": 5432, "portRangeMax": 5433},
	}

	if got, want := DescribeSecurityRules(rules), "0.0.0.0/0:5432, 10.0.0.0/8:5432-5433"; got != want {
		t.Errorf("DescribeSecurityRules = %q, want %q", got, want)
	}
	if got := DescribeSecurityRules(nil); got != "(none)" {
		t.Errorf("empty set = %q, want (none)", got)
	}
}
