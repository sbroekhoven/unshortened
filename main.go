package main

import (
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sbroekhoven/redirects"
)

type config struct {
	Nameserver string
	SiteTitle  string
	SiteURL    string
}

func main() {
	// Load .env file if it exists
	godotenv.Load()

	c := new(config)
	c.SiteTitle = getEnv("SITE_TITLE", "Unshortened")
	c.Nameserver = getEnv("NAMESERVER", "1.1.1.1")
	c.SiteURL = getEnv("SITE_URL", "")
	// Get port from environment with default fallback
	port := getEnv("PORT", "8432")

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		CookieMaxAge:   86400,
		TokenLookup:    "form:csrfmiddlewaretoken",
		CookieHTTPOnly: true,
	}))

	t := &Template{
		templates: template.Must(template.ParseGlob("html/*.html")),
	}
	e.Renderer = t

	e.Static("/static", "static")

	e.GET("/", c.form)
	e.POST("/", c.formPost)

	e.Logger.Fatal(e.Start(":" + port))
}

func (c *config) form(ctx echo.Context) error {
	return ctx.Render(http.StatusOK, "template_bootstrap", map[string]interface{}{
		"pageTitle": c.SiteTitle,
		"response":  nil,
		"error":     nil,
		"csrfToken": ctx.Get(middleware.DefaultCSRFConfig.ContextKey).(string),
	})
}

// TODO: ADD IP information or not?
func (conf *config) formPost(c echo.Context) error {
	url := c.FormValue("url")
	results := redirects.Get(url, conf.Nameserver)

	// do something with the results

	return c.Render(http.StatusOK, "template_bootstrap", map[string]interface{}{
		"PageTitle": conf.SiteTitle,
		"Response":  results,
		"Error":     nil,
		"CSRF":      c.Get(middleware.DefaultCSRFConfig.ContextKey).(string),
	})
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
