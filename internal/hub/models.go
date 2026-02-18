package hub

import "time"

// Index tracks pulled tags and the active tag for a hub pipe.
type Index struct {
	SchemaVersion int                  `json:"schema_version"`
	Owner         string               `json:"owner"`
	Name          string               `json:"name"`
	ActiveTag     string               `json:"active_tag"`
	Tags          map[string]TagRecord `json:"tags"`
}

// TagRecord stores metadata about a pulled tag.
type TagRecord struct {
	SHA256    string    `json:"sha256"`
	MD5       string    `json:"md5"`
	SizeBytes int64     `json:"size_bytes"`
	PulledAt  time.Time `json:"pulled_at,omitzero"`
	CreatedAt time.Time `json:"created_at,omitzero"`
	Editable  bool      `json:"editable,omitempty"`
}

// HeadRef kind constants.
const (
	HeadKindTag  = "tag"
	HeadKindBlob = "blob"
)

// HeadRef describes what HEAD points to: a named tag or an untagged blob.
type HeadRef struct {
	Kind  string // HeadKindTag or HeadKindBlob
	Value string // tag name or sha256 hex
}

// PipeMetadata is the API response for GET /api/v1/pipes/{owner}/{name}.
type PipeMetadata struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	IsMutable   bool   `json:"isMutable"`
}

// TagDetail is the API response for GET /api/v1/pipes/{owner}/{name}/tags/{tag}.
type TagDetail struct {
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
	SHA256    string `json:"sha256"`
	MD5       string `json:"md5"`
	SizeBytes int64  `json:"size_bytes"`
}

// CreatePipeRequest is the body for POST /api/v1/pipes.
type CreatePipeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
}

// PushResponse is the API response after pushing content.
type PushResponse struct {
	Digest    string   `json:"digest"`    // "sha256:<hex>"
	Tags      []string `json:"tags"`
	SizeBytes int64    `json:"sizeBytes"`
	Created   bool     `json:"created"`   // true=new content, false=deduplicated
}
