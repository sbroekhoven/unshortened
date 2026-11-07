package main

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sbroekhoven/redirects"
)

// Templates and static files are built into the binary
//
//go:embed html static
var files embed.FS

type config struct {
	SiteTitle string
	templates *template.Template
}

// pageData holds everything the template uses
type pageData struct {
	PageTitle string
	URL       string
	Response  *redirects.Data
	FinalURL  string
	Error     string
}

func main() {
	// Load .env file if it exists
	godotenv.Load()

	c := new(config)
	c.SiteTitle = getEnv("SITE_TITLE", "Unshortened")
	// Get port from environment with default fallback
	port := getEnv("PORT", "8432")

	c.templates = template.Must(template.New("").Funcs(template.FuncMap{
		"statusText": http.StatusText,
	}).ParseFS(files, "html/*.html"))

	static, err := fs.Sub(files, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", c.form)
	mux.HandleFunc("POST /{$}", c.formPost)
	mux.Handle("GET /static/", http.StripPrefix("/static/", noDirListing(http.FileServerFS(static))))

	// Rejects cross-site form posts, based on the Sec-Fetch-Site and Origin
	// headers that browsers send
	csrf := http.NewCrossOriginProtection()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(logRequests(csrf.Handler(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	slog.Info("starting server", "addr", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func (c *config) form(w http.ResponseWriter, r *http.Request) {
	c.render(w, pageData{})
}

func (c *config) formPost(w http.ResponseWriter, r *http.Request) {
	// The form only has a URL field, so a small body is enough
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	url := r.FormValue("url")
	results := redirects.GetWithOptions(url, redirects.Options{
		BlockPrivateIPs: true,
	})

	data := pageData{
		URL:      url,
		Response: results,
	}
	if results.Error {
		data.Error = results.ErrorMessage
	} else if len(results.Redirects) > 0 {
		data.FinalURL = results.Redirects[len(results.Redirects)-1].URL
	}
	c.render(w, data)
}

// render fills in the fields every page needs and renders the template.
// It renders into a buffer first, so a template error never sends half a page.
func (c *config) render(w http.ResponseWriter, data pageData) {
	data.PageTitle = c.SiteTitle

	var buf bytes.Buffer
	if err := c.templates.ExecuteTemplate(&buf, "template_simple", data); err != nil {
		slog.Error("rendering template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// securityHeaders adds the security headers to every response
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		// Only sent over HTTPS, as reported by the proxy in X-Forwarded-Proto
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// statusRecorder remembers the status code so it can be logged
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// logRequests logs every request with its status and duration
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// noDirListing returns 404 for directories instead of listing their files
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helper function to get environment variable with fallback.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
