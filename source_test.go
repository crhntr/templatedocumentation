package templatedocumentation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crhntr/dom/domtest"
	"github.com/stretchr/testify/assert"
)

func TestSourceHandler(t *testing.T) {
	h := SourceHandler("testdata")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rec, req)

	res := rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	document := domtest.Response(t, res)

	if index := document.QuerySelector(`#index`); assert.NotNil(t, index) {
		links := index.QuerySelectorAll(`a[href*="#"]`)
		for i := 0; i < links.Length(); i++ {
			link := links.Item(i)
			id := strings.TrimPrefix(link.GetAttribute("href"), "#")
			div := document.QuerySelector(fmt.Sprintf("[id=%q]", id))
			if assert.NotNil(t, div, "missing element for %q", id) {
				assert.True(t, div.Matches(`div`), "is a div")
				assert.True(t, div.Matches(`.define-template`), "has class define-template")
				assert.NotNil(t, div.QuerySelector(`pre`), "has source")
			}
		}
	}

	assert.Nil(t, document.QuerySelector(`#index a[href="#template--empty"]`))
	assert.Nil(t, document.QuerySelector(`[id="template--empty"].define-template`))

	if footerSource := document.QuerySelector(`[id="template--footer"]>pre`); assert.NotNil(t, footerSource) {
		assert.NotNil(t, footerSource.QuerySelector(`a[href="#template--nav"]`))
	}

	if t.Failed() && testing.Verbose() {
		t.Log(document)
	}
}
