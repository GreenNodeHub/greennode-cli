package vks

import (
	"net/url"
	"testing"
)

func TestListNodegroupImagesWire(t *testing.T) {
	assertVKSReadWire(t, listNodegroupImagesCmd, nil, "/v1/node-group-images", url.Values{}, `[{"id":"fixture-result","os":"ubuntu","kubernetesVersion":"fixture-version","enable":true}]`, "[0].id")
}
