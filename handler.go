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

func Handler(templates *template.Template, functions template.FuncMap) http.Handler {
	srv := &server{templates: templates, functions: functions}
	return http.HandlerFunc(srv.page)
}

type server struct {
	templates *template.Template
	functions template.FuncMap
}

func (srv *server) TemplateLinks() []Link {
	var links []Link
	for _, ts := range srv.templates.Templates() {
		if isEmptyTemplate(ts) {
			continue
		}
		links = append(links, newTemplateLink(ts))
	}
	slices.SortFunc(links, func(a, b Link) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return slices.Clip(links)
}

func isEmptyTemplate(ts *template.Template) bool {
	name := ts.Name()
	return name == "" || ts.Tree == nil || ts.Tree.Root == nil || strings.TrimSpace(ts.Tree.Root.String()) == ""
}

func (srv *server) page(res http.ResponseWriter, req *http.Request) {
	render(res, req, http.StatusOK, "page", srv)
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

func (srv *server) FunctionLinks() []Link {
	var links []Link
	for name, function := range srv.functions {
		if name == "" {
			continue
		}
		links = append(links, newFunctionLink(name, function))
	}
	slices.SortFunc(links, func(a, b Link) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return slices.Clip(links)
}

func (srv *server) Templates() []Template {
	var result []Template
	for _, ts := range srv.templates.Templates() {
		if isEmptyTemplate(ts) {
			continue
		}
		result = append(result, Template{Template: ts, functions: srv.functions})
	}
	return result

}

type Link struct {
	Name      string
	SafeID    string
	Signature string
}

func newLink(prefix, name string) Link {
	return Link{
		Name:   name,
		SafeID: templateID(name),
	}
}

func templateID(name string) string {
	return "template--" + url.QueryEscape(name)
}

func newFunctionLink(name string, anyFunction any) Link {
	link := newLink("function--", name)

	function := reflect.ValueOf(anyFunction)
	fnType := strings.TrimPrefix(function.Type().String(), "func")
	fnType = strings.Replace(fnType, "interface {}", "any", -1)

	link.Name = "func " + link.Name + fnType
	return link
}

func newTemplateLink(template *template.Template) Link {
	link := newLink("function--", template.Name())
	link.Name = "{{template " + strconv.Quote(link.Name) + " . }}"
	return link
}

type Template struct {
	*template.Template
	functions template.FuncMap
}

func (ts Template) ID() string {
	return templateID(ts.Template.Name())
}

func (ts Template) Definition() template.HTML {
	if ts.Template == nil ||
		ts.Template.Tree == nil ||
		ts.Template.Tree.Root == nil {
		return ""
	}
	definition := html.EscapeString(ts.Template.Tree.Root.String())

	templateExp := regexp.MustCompile(`(?mU)\{\{template\s+&#34;(.+)&#34;\s+.*}}`)
	matchExp := regexp.MustCompile(`(?mU)&#34;(.*)&#34;`)

	replaced := templateExp.ReplaceAllStringFunc(definition, func(match string) string {
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
