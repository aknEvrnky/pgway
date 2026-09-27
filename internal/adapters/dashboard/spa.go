package dashboard

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/aknEvrnky/pgway/internal/adapters/dashboard/ui"
)

// withSPA serves API routes from mux and falls back to the embedded SPA for
// non-/api GET/HEAD requests. Using a wrapper avoids ServeMux conflicts between
// method-specific API patterns and a GET/HEAD /{path...} catch-all.
func (a *Adapter) withSPA(api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			a.serveSPA(w, r)
		default:
			api.ServeHTTP(w, r)
		}
	})
}

// contentFS returns the static UI filesystem when available.
// staticFS is an optional test override.
func (a *Adapter) contentFS() (fs.FS, bool) {
	if a.staticFS != nil {
		return a.staticFS, true
	}
	if ui.Enabled() {
		return ui.FS, true
	}
	return nil, false
}

// serveSPA serves embedded (or test) static assets with index.html fallback
// for client-side routes.
func (a *Adapter) serveSPA(w http.ResponseWriter, r *http.Request) {
	fsys, ok := a.contentFS()
	if !ok {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || strings.HasSuffix(name, "/") {
		name = "index.html"
	}

	if _, err := fs.Stat(fsys, name); err != nil {
		name = "index.html"
		if _, err := fs.Stat(fsys, name); err != nil {
			http.NotFound(w, r)
			return
		}
	}

	http.ServeFileFS(w, r, fsys, name)
}
