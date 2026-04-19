package semantic

import "time"

const (
	DefaultUri       = "http://semantic:8000"
	ServiceTimeout   = 30 * time.Second
	maxResponseBytes = 4 << 20 // 4 MB
)

// Config holds configuration for the semantic search sidecar.
type Config struct {
	Enabled bool
	Rerank  bool
	Uri     string
}

// IsEnabled returns true if the semantic search service is configured and enabled.
func (c Config) IsEnabled() bool {
	return c.Enabled && c.Uri != ""
}
