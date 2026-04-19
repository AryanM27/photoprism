package config

import (
	"github.com/photoprism/photoprism/internal/ai/semantic"
	"github.com/photoprism/photoprism/pkg/clean"
)

// SemanticConfig returns the semantic search sidecar configuration.
func (c *Config) SemanticConfig() semantic.Config {
	if c == nil {
		return semantic.Config{}
	}

	conf := semantic.Config{
		Enabled: c.options.SemanticEnabled,
		Rerank:  c.options.SemanticRerank,
		Uri:     clean.Uri(c.options.SemanticUri),
	}

	if conf.Enabled && conf.Uri == "" {
		conf.Uri = semantic.DefaultUri
	}

	return conf
}
