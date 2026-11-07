package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sbroekhoven/redirects"
	"golang.org/x/time/rate"
)

// Templates and static files are built into the binary
//
//go:embed html static
var files embed.FS

type config struct {
	SiteTitle string
	SiteURL   string
}

// pageData holds everything the template uses
type pageData struct {
	PageTitle string
	URL       string
	Response  *redirects.Data
	FinalURL  string
	Error     string
	CSRF      string
}

func main() {
	// Load .env file if it exists
	godotenv.Load()

	c := new(config)
	c.SiteTitle = getEnv("SITE_TITLE", "Unshortened")
	c.SiteURL = getEnv("SITE_URL", "")
	// Get port from environment with default fallback
	port := getEnv("PORT", "8432")

	e := echo.New()
	e.Server.ReadHeaderTimeout = 10 * time.Second
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.IdleTimeout = 2 * time.Minute

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "0",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'self'; img-src 'self' data:; base-uri 'self'; form-action 'self'; frame-ancestors 'none'",
	}))
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		CookieMaxAge:   86400,
		TokenLookup:    "form:csrfmiddlewaretoken",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
	}))

	t := &Template{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"statusText": http.StatusText,
		}).ParseFS(files, "html/*.html")),
	}
	e.Renderer = t

	// Each lookup makes outgoing requests, so limit lookups per visitor
	limiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(10.0 / 60), // 10 per minute
			Burst:     5,
			ExpiresIn: 5 * time.Minute,
		},
	))

	e.StaticFS("/static", echo.MustSubFS(files, "static"))
	e.GET("/", c.form)
	e.POST("/", c.formPost, limiter)

	e.Logger.Fatal(e.Start(":" + port))
}

func (c *config) form(ctx echo.Context) error {
	return c.render(ctx, "", nil)
}

func (c *config) formPost(ctx echo.Context) error {
	url := ctx.FormValue("url")
	results := redirects.GetWithOptions(url, redirects.Options{
		BlockPrivateIPs: true,
	})
	return c.render(ctx, url, results)
}

func (c *config) render(ctx echo.Context, url string, results *redirects.Data) error {
	data := pageData{
		PageTitle: c.SiteTitle,
		URL:       url,
		Response:  results,
		CSRF:      ctx.Get(middleware.DefaultCSRFConfig.ContextKey).(string),
	}

	if results != nil {
		if results.Error {
			data.Error = results.ErrorMessage
		} else if len(results.Redirects) > 0 {
			data.FinalURL = results.Redirects[len(results.Redirects)-1].URL
		}
	}

	return ctx.Render(http.StatusOK, "template_simple", data)
}

// Template struct for HTML template rendering
type Template struct {
	templates *template.Template
}

// Render function for templates
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Helper function to get environment variable with fallback.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
