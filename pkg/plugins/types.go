package plugins

// Manifest describes a dashboard Web Component plugin the host may load.
// Dashboard panel JSON references only ID (+ optional Version); never free-form URLs.
type Manifest struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
	Element     string `json:"element"`
	// Source is "catalog" for disk-served bundles (frontend also has "builtin").
	Source      string `json:"source"`
	BundlePath  string `json:"bundlePath,omitempty"`
	Description string `json:"description,omitempty"`
}

// CatalogFile is the on-disk shape of plugins/manifest.json.
type CatalogFile struct {
	Plugins []Manifest `json:"plugins"`
}
