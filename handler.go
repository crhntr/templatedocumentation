package templatedocumentation

import (
	"bytes"
	"cmp"
	_ "embed"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
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

func isEmptyTemplate(ts *template.Template) bool {
	name := ts.Name()
	return name == "" || ts.Tree == nil || ts.Tree.Root == nil || strings.TrimSpace(ts.Tree.Root.String()) == ""
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
		if isEmptyTemplate(ts) {
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
	var result []definition
	for _, ts := range pg.templates.Templates() {
		if isEmptyTemplate(ts) {
			continue
		}
		result = append(result, definition{Template: ts, functions: pg.functions})
	}
	return result
}

type link struct {
	Name      string
	SafeID    string
	Signature string
}

func newLink(prefix, name string) link {
	return link{
		Name:   name,
		SafeID: templateID(name),
	}
}

func templateID(name string) string {
	return "template--" + url.QueryEscape(name)
}

func newFunctionLink(name string, anyFunction any) link {
	link := newLink("function--", name)

	function := reflect.ValueOf(anyFunction)
	fnType := strings.TrimPrefix(function.Type().String(), "func")
	fnType = strings.Replace(fnType, "interface {}", "any", -1)

	link.Name = "func " + link.Name + fnType
	return link
}

func newTemplateLink(template *template.Template) link {
	link := newLink("function--", template.Name())
	link.Name = "{{template " + strconv.Quote(link.Name) + " . }}"
	return link
}

type definition struct {
	*template.Template
	functions template.FuncMap
}

func (ts definition) ID() string {
	return templateID(ts.Template.Name())
}

func (ts definition) Definition() template.HTML {
	if ts.Template == nil ||
		ts.Template.Tree == nil ||
		ts.Template.Tree.Root == nil {
		return ""
	}
	src := html.EscapeString(ts.Template.Tree.Root.String())

	templateExp := regexp.MustCompile(`(?mU)\{\{template\s+&#34;(.+)&#34;\s+.*}}`)
	matchExp := regexp.MustCompile(`(?mU)&#34;(.*)&#34;`)

	replaced := templateExp.ReplaceAllStringFunc(src, func(match string) string {
		matches := matchExp.FindStringSubmatch(match)
		if len(matches) <= 1 {
			return match
		}
		name := matches[1]
		var buf bytes.Buffer
		if err := templates.ExecuteTemplate(&buf, "template_link", struct {
			Link   string
			Source string
		}{
			Link:   templateID(name),
			Source: html.UnescapeString(match),
		}); err != nil {
			return match
		}
		return buf.String()
	})

	return template.HTML(replaced)
}
