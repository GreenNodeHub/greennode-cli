package vks

import (
	"net/url"
	"strconv"
	"testing"
)

func TestGetUpgradeInsightsWire(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(strconv.FormatBool(explicit), func(t *testing.T) {
			flags := map[string]string{"cluster-id": "fixture-cluster"}
			query := url.Values{}
			if explicit {
				flags["page"] = "0"
				flags["page-size"] = "10"
				query = url.Values{"page": {"0"}, "pageSize": {"10"}}
			}
			assertVKSReadWire(t, getUpgradeInsightsCmd, flags, "/v1/clusters/fixture-cluster/upgrade-insight", query, `{"items":[{"id":"fixture-result"}],"page":0,"pageSize":10,"total":1}`, "items[0].id")
		})
	}
}
