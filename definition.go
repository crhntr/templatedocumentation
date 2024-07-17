package templatedocumentation

import (
	"bytes"
	"cmp"
	"html"
	"html/template"
	"regexp"
	"slices"
	"strings"
	"text/template/parse"
)

type definition struct {
	tree *parse.Tree
}

func definitionsFromTreeSet(ts treeSet) []definition {
	var result []definition
	for _, t := range ts {
		if isEmptyTree(t) {
			continue
		}
		result = append(result, definition{tree: t})
	}
	sortDefinitions(result)
	return result
}

func definitionsFromTemplates(ts []*template.Template) []definition {
	var result []definition
	for _, t := range ts {
		if isEmptyTree(t.Tree) {
			continue
		}
		result = append(result, definition{tree: t.Tree})
	}
	sortDefinitions(result)
	return result
}

func sortDefinitions(definitions []definition) {
	slices.SortFunc(definitions, func(a, b definition) int {
		return cmp.Compare(a.tree.Name, b.tree.Name)
	})
}

func isEmptyTree(ts *parse.Tree) bool {
	return ts == nil || ts.Name == "" || ts.Root == nil || strings.TrimSpace(ts.Root.String()) == ""
}

func (ts definition) Name() string {
	if ts.tree == nil {
		return ""
	}
	return ts.tree.Name
}

func (ts definition) ID() string {
	return identifier(templatePrefix, ts.Name())
}

func (ts definition) SourceHTML() template.HTML {
	if ts.tree == nil ||
		ts.tree.Root == nil {
		return ""
	}
	src := html.EscapeString(ts.tree.Root.String())

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
			Link:   identifier(templatePrefix, name),
			Source: html.UnescapeString(match),
		}); err != nil {
			return match
		}
		return buf.String()
	})

	return template.HTML(replaced)
}
