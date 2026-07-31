package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	idVersionRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
)

// Catalog is a thread-safe allowlist of plugins and their on-disk bundles.
type Catalog struct {
	mu        sync.RWMutex
	dir       string
	enabled   bool
	byKey     map[string]Manifest // id@version
	byID      map[string][]Manifest
}

// NewCatalog loads manifest.json from dir. Missing dir/manifest yields an empty
// but usable catalog (API returns []).
func NewCatalog(cfg Config) (*Catalog, error) {
	c := &Catalog{
		dir:     cfg.Dir,
		enabled: cfg.Enabled,
		byKey:   make(map[string]Manifest),
		byID:    make(map[string][]Manifest),
	}
	if !cfg.Enabled {
		return c, nil
	}
	if err := c.Reload(); err != nil {
		// Non-fatal when the directory simply does not exist yet.
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	return c, nil
}

// Reload re-reads manifest.json from disk.
func (c *Catalog) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := filepath.Join(c.dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var file CatalogFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse plugin manifest: %w", err)
	}

	byKey := make(map[string]Manifest, len(file.Plugins))
	byID := make(map[string][]Manifest)
	for _, m := range file.Plugins {
		if err := validateManifest(m); err != nil {
			return fmt.Errorf("plugin %q: %w", m.ID, err)
		}
		if m.Source == "" {
			m.Source = "catalog"
		}
		if m.Version == "" {
			m.Version = "latest"
		}
		// Canonical API bundle path if author omitted one.
		if m.BundlePath == "" {
			m.BundlePath = fmt.Sprintf("/api/v1/plugins/%s/%s/bundle.js", m.ID, m.Version)
		}
		key := m.ID + "@" + m.Version
		byKey[key] = m
		byID[m.ID] = append(byID[m.ID], m)
	}
	c.byKey = byKey
	c.byID = byID
	return nil
}

func validateManifest(m Manifest) error {
	if m.ID == "" || !idVersionRe.MatchString(m.ID) {
		return fmt.Errorf("invalid id %q", m.ID)
	}
	if m.Version != "" && !idVersionRe.MatchString(m.Version) {
		return fmt.Errorf("invalid version %q", m.Version)
	}
	if m.Element == "" {
		return fmt.Errorf("element is required")
	}
	// Element must look like a custom element (contains a hyphen).
	if !strings.Contains(m.Element, "-") {
		return fmt.Errorf("element %q must contain a hyphen (custom element)", m.Element)
	}
	return nil
}

// Enabled reports whether the plugin API is on.
func (c *Catalog) Enabled() bool {
	if c == nil {
		return false
	}
	return c.enabled
}

// List returns all registered manifests.
func (c *Catalog) List() []Manifest {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Manifest, 0, len(c.byKey))
	for _, m := range c.byKey {
		out = append(out, m)
	}
	return out
}

// Get returns a manifest by id and optional version. Empty version picks the first registered.
func (c *Catalog) Get(id, version string) (Manifest, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if version != "" {
		m, ok := c.byKey[id+"@"+version]
		return m, ok
	}
	list := c.byID[id]
	if len(list) == 0 {
		return Manifest{}, false
	}
	return list[len(list)-1], true
}

// BundleFile returns the absolute path to the on-disk bundle for id@version.
// Only paths under the catalog dir are returned (path-traversal safe).
func (c *Catalog) BundleFile(id, version string) (string, error) {
	if !idVersionRe.MatchString(id) || !idVersionRe.MatchString(version) {
		return "", fmt.Errorf("invalid plugin id or version")
	}
	if _, ok := c.Get(id, version); !ok {
		return "", fmt.Errorf("plugin not found")
	}
	// Prefer plugin.js; also accept bundle.js for author convenience.
	candidates := []string{
		filepath.Join(c.dir, id, version, "plugin.js"),
		filepath.Join(c.dir, id, version, "bundle.js"),
	}
	root, err := filepath.Abs(c.dir)
	if err != nil {
		return "", err
	}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path escapes plugin directory")
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf("bundle file not found for %s@%s", id, version)
}
