package cache

import (
	"strings"
	"time"
)

// normalizedKeyPrefix preserves unprefixed keys when no namespace is configured.
func normalizedKeyPrefix(prefixes ...string) string {
	if len(prefixes) == 0 || prefixes[0] == "" {
		return ""
	}
	prefix := prefixes[0]
	if !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	return prefix
}

const (
	KeyPrefixModuleData          = "module:data:"
	KeyPrefixScriptBundle        = "script:bundle:"
	KeyPrefixVideoStatus         = "video:status:"
	KeyPrefixChartRuntimeBinding = "chart:runtime:binding:"
	KeyPrefixChartRuntimeChart   = "chart:runtime:chart:"
	KeyCurrentScript             = "script:current"
)

const (
	DefaultModuleTTL       = 30 * time.Second
	DefaultScriptTTL       = 60 * time.Second
	DefaultVideoTTL        = 30 * time.Second
	DefaultChartRuntimeTTL = 60 * time.Second
)
