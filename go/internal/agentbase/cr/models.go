// Package cr is the typed client for the agent-core-container-registry service
// (`grn agentbase cr`). The service wraps VNG Cloud Container Registry (vCR),
// auto-provisioning a per-user repository + robot account so end users need not
// touch vCR directly.
//
// agent-core-container-registry is a Go/Gin service; only the wire contract
// matters here. Auth is the shared seam — Bearer → IAM ingress →
// `portal-user-id` header (int64). The /api/v1 group runs a provisioning
// middleware that creates the user's repo + robot account on first access, so
// there is no create endpoint — resources appear when first read.
//
// Wire shape notes (load-bearing — a FOURTH distinct paging envelope):
//   - Paths are versioned: `/api/v1/repository[/{images,artifacts}]` and
//     `/api/v1/registry-credential[/secret]`.
//   - List envelope is `{data, page, pageSize, totalItem, totalPage}` — items
//     under `data` (not items/listData/content). Query: ?page=&size=&name=
//     (1-based; size clamped to 100). Artifacts list additionally requires
//     ?imageName=.
//   - Deletes identify the target by QUERY params (?imageName=, ?digest=), not
//     a path segment or body, and return 204 No Content.
//   - reset-secret is PATCH /api/v1/registry-credential/secret (no body).
//   - Registry credentials (robot account) expose {username, secret}; the secret
//     is real and used for docker login — mask it in table output, reveal in JSON.
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
package cr

import "time"

// Repository is the user's auto-provisioned vCR repository info.
type Repository struct {
	Name        string    `json:"name"`
	RegistryURL string    `json:"registryUrl"`
	ImageCount  int32     `json:"imageCount"`
	QuotaUsed   int64     `json:"quotaUsed"`
	QuotaLimit  int64     `json:"quotaLimit"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Image is one container image (a vCR repository) in the user's namespace.
type Image struct {
	Name          string    `json:"name"`
	ArtifactCount int32     `json:"artifactCount"`
	PullCount     int32     `json:"pullCount"`
	UpdateTime    time.Time `json:"updateTime"`
}

// Tag is one tag on an artifact.
type Tag struct {
	Name     string     `json:"name"`
	PushTime time.Time  `json:"pushTime"`
	PullTime *time.Time `json:"pullTime"`
}

// Artifact is one pushable unit (by digest) of an image, carrying its tags.
type Artifact struct {
	Digest   string     `json:"digest"`
	Type     string     `json:"type"`
	Size     int64      `json:"size"`
	Tags     []Tag      `json:"tags"`
	PushTime time.Time  `json:"pushTime"`
	PullTime *time.Time `json:"pullTime"`
}

// RegistryCredential is the user's robot account (username + secret) for
// programmatic registry push/pull. The secret is real — callers use it for
// `docker login`. Handle with care: do not log it.
type RegistryCredential struct {
	Username string `json:"username"`
	Secret   string `json:"secret"`
}

// ListImagesResponse is the body of GET /api/v1/repository/images. Items live
// under `data`; paging is page/pageSize/totalItem/totalPage.
type ListImagesResponse struct {
	Data      []Image `json:"data"`
	Page      int32   `json:"page"`
	PageSize  int32   `json:"pageSize"`
	TotalItem int64   `json:"totalItem"`
	TotalPage int32   `json:"totalPage"`
}

// ListArtifactsResponse is the body of GET /api/v1/repository/artifacts. Same
// `{data, …}` envelope.
type ListArtifactsResponse struct {
	Data      []Artifact `json:"data"`
	Page      int32      `json:"page"`
	PageSize  int32      `json:"pageSize"`
	TotalItem int64      `json:"totalItem"`
	TotalPage int32      `json:"totalPage"`
}
