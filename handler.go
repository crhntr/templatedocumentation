package templatedocumentation

import (
	"bytes"
	"cmp"
	_ "embed"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template/parse"
)

var (
	//go:embed template.gohtml
	templateGoHTML string

	templates = template.Must(template.New("documentation").Parse(templateGoHTML))
)

type TemplatesFunc func() (*template.Template, template.FuncMap, error)

func Handler(templates TemplatesFunc) http.Handler {
	srv := &server{templates: templates}
	return http.HandlerFunc(srv.page)
}

type server struct {
	templates TemplatesFunc
}

func isEmptyTemplate(ts *parse.Tree) bool {
	return ts == nil || ts.Name == "" || ts.Root == nil || strings.TrimSpace(ts.Root.String()) == ""
}

func render(res http.ResponseWriter, _ *http.Request, code int, name string, data any) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	header := res.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Length", strconv.Itoa(buf.Len()))
	res.WriteHeader(code)
	_, _ = buf.WriteTo(res)
}

func (srv *server) page(res http.ResponseWriter, req *http.Request) {
	ts, fns, err := srv.templates()
	render(res, req, http.StatusOK, "page", indexPage{
		err:       err,
		templates: ts,
		functions: fns,
	})
}

type indexPage struct {
	templates *template.Template
	functions template.FuncMap
	err       error
}

func (pg indexPage) TemplateLinks() []link {
	var links []link
	for _, ts := range pg.templates.Templates() {
		if ts == nil || isEmptyTemplate(ts.Tree) {
			continue
		}
		links = append(links, newTemplateLink(ts))
	}
	slices.SortFunc(links, func(a, b link) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return slices.Clip(links)
}

func (pg indexPage) FunctionLinks() []link {
	var links []link
	for name, function := range pg.functions {
		if name == "" {
			continue
		}
		links = append(links, newFunctionLink(name, function))
	}
	slices.SortFunc(links, func(a, b link) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return slices.Clip(links)
}

func (pg indexPage) Templates() []definition {
	return definitionsFromTemplates(pg.templates.Templates())
}

type link struct {
	Name      string
	SafeID    string
	Signature string
}

const (
	functionPrefix = "function--"
	templatePrefix = "template--"
)

func newLink(prefix, name string) link {
	return link{
		Name:   name,
		SafeID: identifier(prefix, name),
	}
}

func identifier(prefix, name string) string {
	return prefix + url.QueryEscape(name)
}

func newFunctionLink(name string, anyFunction any) link {
	a := newLink(functionPrefix, name)

	function := reflect.ValueOf(anyFunction)
	fnType := strings.TrimPrefix(function.Type().String(), "func")
	fnType = strings.Replace(fnType, "interface {}", "any", -1)

	a.Name = "func " + a.Name + fnType
	return a
}

func newTemplateLink(template *template.Template) link {
	a := newLink(templatePrefix, template.Name())
	a.Name = "{{template " + strconv.Quote(a.Name) + " . }}"
	return a
}
