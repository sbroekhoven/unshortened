package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sbro101/goredirects"
)

type config struct {
	Nameserver string
	SiteTitle  string
	SiteURL    string
}

func main() {
	conf := new(config)
	conf.SiteTitle = "Unshortened"
	conf.Nameserver = "1.1.1.1"

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CSRF
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		CookieMaxAge: 86400,
		TokenLookup:  "form:csrfmiddlewaretoken",
		// CookieSecure:   true,
		CookieHTTPOnly: true,
		// Skipper:      DefaultSkipper,
		// TokenLength:  32,
		// TokenLookup:  "header:" + echo.HeaderXCSRFToken,
		// ContextKey:   "csrf",
		// CookieName:   "_csrf",
	}))

	// HTML templatesS
	t := &Template{
		templates: template.Must(template.ParseGlob("html/*.html")),
	}
	e.Renderer = t

	e.Static("/static", "static")

	// Routes
	e.GET("/", conf.form)
	e.POST("/", conf.formPost)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

func (conf *config) form(c echo.Context) error {
	return c.Render(http.StatusOK, "template_bootstrap", map[string]interface{}{
		"PageTitle": conf.SiteTitle,
		"Response":  nil,
		"Error":     nil,
		"CSRF":      c.Get(middleware.DefaultCSRFConfig.ContextKey).(string),
	})
}

// TODO: ADD IP information
func (conf *config) formPost(c echo.Context) error {
	url := c.FormValue("url")
	results := goredirects.Get(url, conf.Nameserver)

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
