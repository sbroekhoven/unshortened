package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sbro101/redirects"
)

type config struct {
	Nameserver string
	SiteTitle  string
	SiteURL    string
}

func main() {
	c := new(config)
	c.SiteTitle = "Unshortened"
	c.Nameserver = "1.1.1.1"

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

	e.Logger.Fatal(e.Start(":1323"))
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
