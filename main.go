package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
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

const csrfContextKey = "csrf"

const csrfErrorMessage = "Your session expired or the request came from another site. Please try again."

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
		// Only sent over HTTPS, as reported by the proxy in X-Forwarded-Proto
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: true,
	}))
	// Headers that the Secure middleware doesn't support
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			h := ctx.Response().Header()
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			return next(ctx)
		}
	})
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		// Static files don't need a CSRF cookie
		Skipper: func(ctx echo.Context) bool {
			return strings.HasPrefix(ctx.Request().URL.Path, "/static/")
		},
		ContextKey:     csrfContextKey,
		CookieMaxAge:   86400,
		TokenLookup:    "form:csrfmiddlewaretoken",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		// Only mark the cookie Secure when the site runs on HTTPS, so local
		// use over plain HTTP keeps working
		CookieSecure: strings.HasPrefix(c.SiteURL, "https://"),
		// Redirect so the page is loaded again with a fresh token
		ErrorHandler: func(err error, ctx echo.Context) error {
			return ctx.Redirect(http.StatusSeeOther, "/?error=csrf")
		},
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
	data := pageData{}
	if ctx.QueryParam("error") == "csrf" {
		data.Error = csrfErrorMessage
	}
	return c.render(ctx, data)
}

func (c *config) formPost(ctx echo.Context) error {
	url := ctx.FormValue("url")
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
	return c.render(ctx, data)
}

// render fills in the fields every page needs and renders the template
func (c *config) render(ctx echo.Context, data pageData) error {
	data.PageTitle = c.SiteTitle
	data.CSRF, _ = ctx.Get(csrfContextKey).(string)

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
