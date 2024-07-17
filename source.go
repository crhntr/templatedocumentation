package templatedocumentation

import (
	"cmp"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"text/template/parse"
)

func SourceHandler(directory string) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		set := make(treeSet)
		for _, pattern := range []string{
			filepath.Join(directory, "*.gohtml"),
		} {
			filePaths, err := filepath.Glob(pattern)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, filePath := range filePaths {
				if err = parseSource(set, filePath); err != nil {
					http.Error(res, err.Error(), http.StatusBadRequest)
					return
				}
			}
		}
		render(res, req, http.StatusOK, "page", newSourceIndex(set))
	})
}

type sourceIndex struct {
	set treeSet
}

func newSourceIndex(set treeSet) *sourceIndex {
	return &sourceIndex{
		set: set,
	}
}

func (data *sourceIndex) TemplateLinks() []link {
	links := make([]link, 0, len(data.set))
	for _, tree := range data.set {
		if isEmptyTemplate(tree) {
			continue
		}
		a := newLink(templatePrefix, tree.Name)
		a.Name = "{{template " + strconv.Quote(a.Name) + " . }}"
		links = append(links, a)
	}
	slices.SortFunc(links, func(a, b link) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return links
}

func (data *sourceIndex) FunctionLinks() []link {
	result := make(map[string][]parse.Node)
	for _, tree := range data.set {
		listFunctions(tree, result)
	}
	links := make([]link, 0, len(result))
	for name := range result {
		l := newLink(functionPrefix, name)
		links = append(links, l)
	}
	return links
}

func (data *sourceIndex) Templates() []definition {
	return definitionsFromTreeSet(data.set)
}

type treeSet map[string]*parse.Tree

func parseSource(set treeSet, filePath string) error {
	buf, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	filePath = filepath.ToSlash(filePath)
	fileName := path.Base(filePath)
	tree := parse.New(fileName)
	tree.Mode |= parse.SkipFuncCheck | parse.ParseComments
	tree, err = tree.Parse(string(buf), "{{", "}}", set)
	if err != nil {
		return err
	}
	return nil
}

func listFunctions(node *parse.Tree, functions map[string][]parse.Node) {
	for _, n := range node.Root.Nodes {
		switch nd := n.(type) {
		case *parse.ActionNode:
			for _, cmd := range nd.Pipe.Cmds {
				for _, id := range cmd.Args {
					if id.Type() != parse.NodeIdentifier {
						continue
					}
					functions[id.String()] = append(functions[id.String()], id)
				}
			}
		}
	}
}
