package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogLoadAndGet(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
		"plugins": [
			{
				"id": "devops-status",
				"version": "1.0.0",
				"displayName": "DevOps Status",
				"element": "lax-devops-status"
			}
		]
	}`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "devops-status", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "devops-status", "1.0.0", "plugin.js"),
		[]byte("/* ok */"),
		0o644,
	))

	cat, err := NewCatalog(Config{Enabled: true, Dir: dir})
	require.NoError(t, err)

	list := cat.List()
	require.Len(t, list, 1)
	assert.Equal(t, "devops-status", list[0].ID)
	assert.Equal(t, "catalog", list[0].Source)
	assert.Contains(t, list[0].BundlePath, "/api/v1/plugins/devops-status/1.0.0/bundle.js")

	m, ok := cat.Get("devops-status", "1.0.0")
	require.True(t, ok)
	assert.Equal(t, "lax-devops-status", m.Element)

	path, err := cat.BundleFile("devops-status", "1.0.0")
	require.NoError(t, err)
	assert.FileExists(t, path)
}

func TestCatalogRejectsBadID(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
		"plugins": [{ "id": "../etc", "version": "1", "element": "x-y" }]
	}`), 0o644))

	_, err := NewCatalog(Config{Enabled: true, Dir: dir})
	require.Error(t, err)
}

func TestCatalogRejectsPathTraversalOnBundle(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
		"plugins": [{
			"id": "hello",
			"version": "1.0.0",
			"displayName": "Hello",
			"element": "lax-hello-wc"
		}]
	}`), 0o644))
	cat, err := NewCatalog(Config{Enabled: true, Dir: dir})
	require.NoError(t, err)

	_, err = cat.BundleFile("hello", "1.0.0")
	require.Error(t, err) // no file on disk

	_, err = cat.BundleFile("../etc", "passwd")
	require.Error(t, err)
}

func TestCatalogDisabled(t *testing.T) {
	cat, err := NewCatalog(Config{Enabled: false, Dir: "/nope"})
	require.NoError(t, err)
	assert.False(t, cat.Enabled())
	assert.Empty(t, cat.List())
}

func TestCatalogMissingDir(t *testing.T) {
	cat, err := NewCatalog(Config{Enabled: true, Dir: filepath.Join(t.TempDir(), "missing")})
	require.NoError(t, err)
	assert.Empty(t, cat.List())
}
