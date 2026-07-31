package plugins

import (
	"net/http"
	"os"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/gorilla/mux"
)

// Handler exposes the plugin catalog over HTTP.
type Handler struct {
	catalog *Catalog
}

func NewHandler(catalog *Catalog) *Handler {
	return &Handler{catalog: catalog}
}

// List GET /api/v1/plugins
func (h *Handler) List(rw http.ResponseWriter, r *http.Request) {
	if h.catalog == nil || !h.catalog.Enabled() {
		render.Success(rw, http.StatusOK, []Manifest{})
		return
	}
	render.Success(rw, http.StatusOK, h.catalog.List())
}

// Get GET /api/v1/plugins/{id}
// Optional query: ?version=
func (h *Handler) Get(rw http.ResponseWriter, r *http.Request) {
	if h.catalog == nil || !h.catalog.Enabled() {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "plugins disabled"))
		return
	}
	id := mux.Vars(r)["id"]
	version := r.URL.Query().Get("version")
	m, ok := h.catalog.Get(id, version)
	if !ok {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "plugin %q not found", id))
		return
	}
	render.Success(rw, http.StatusOK, m)
}

// Bundle GET /api/v1/plugins/{id}/{version}/bundle.js
// Serves the allowlisted on-disk JS file only — never arbitrary paths.
func (h *Handler) Bundle(rw http.ResponseWriter, r *http.Request) {
	if h.catalog == nil || !h.catalog.Enabled() {
		http.NotFound(rw, r)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	version := vars["version"]
	path, err := h.catalog.BundleFile(id, version)
	if err != nil {
		http.NotFound(rw, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(rw, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.NotFound(rw, r)
		return
	}
	rw.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	rw.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeContent(rw, r, "bundle.js", st.ModTime(), f)
}

// AuthWrap is the shape of AuthZ access wrappers (ViewAccess, OpenAccess, …).
type AuthWrap func(http.HandlerFunc) http.HandlerFunc

// RegisterRoutes attaches plugin routes.
// Bundle is open-access for classic <script> tags; the catalog allowlist is the security boundary.
func RegisterRoutes(router *mux.Router, h *Handler, view, open AuthWrap) {
	if h == nil {
		return
	}
	router.HandleFunc("/api/v1/plugins", view(h.List)).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/plugins/{id}", view(h.Get)).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/plugins/{id}/{version}/bundle.js", open(h.Bundle)).Methods(http.MethodGet)
}
