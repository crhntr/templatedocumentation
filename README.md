# Generate HTTP Template Documentation for your Project [![Go Reference](https://pkg.go.dev/badge/github.com/templatedocumentation/dom.svg)](https://pkg.go.dev/github.com/crhntr/templatedocumentation)

## Usage

1. Create a main package to serve the documentation

   `mkdir -p ./cmd/template-docs`


2. Create a `main.go` file in the `./cmd/template-docs` directory

   ```go
   package main

   import (
      "cmp"
      _ "embed"
      "html/template"
      "log"
      "net/http"
      "os"

       "github.com/crhntr/templatedocumentation"

       "example.com/org/module/internal/hypertext" // replace with your module
   )

   func main() {
      h := templatedocumentation.Handler(func() (*template.Template, template.FuncMap, error) {
         fn := hypertext.Functions()
         ts, err := hypertext.Template()
         return ts, fn, err
      })
      log.Fatal(http.ListenAndServe(":"+cmp.Or(os.Getenv("DOCS_PORT"), "8200"), h))
   }
   ```

3. Run the server

   `go run ./cmd/template-docs`


4. Navigate to `http://localhost:8200` to view the documentation.