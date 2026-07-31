package plugins

import "os"

// Config controls where plugin manifests and bundles are loaded from.
type Config struct {
	// Enabled toggles the plugin catalog API. Default true.
	Enabled bool `mapstructure:"enabled"`
	// Dir is the root directory containing manifest.json and plugin bundles.
	// Layout:
	//   {Dir}/manifest.json
	//   {Dir}/{id}/{version}/plugin.js
	Dir string `mapstructure:"dir"`
}

// NewConfigFromEnv builds config with defaults, overridden by env when set.
// SIGNOZ_PLUGINS_DIR — plugin root (default: ./plugins)
// SIGNOZ_PLUGINS_ENABLED — "false" to disable (default: enabled)
func NewConfigFromEnv() Config {
	cfg := Config{
		Enabled: true,
		Dir:     "./plugins",
	}
	if v := os.Getenv("SIGNOZ_PLUGINS_DIR"); v != "" {
		cfg.Dir = v
	}
	if v := os.Getenv("SIGNOZ_PLUGINS_ENABLED"); v == "false" || v == "0" {
		cfg.Enabled = false
	}
	return cfg
}
