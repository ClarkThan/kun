package httptest

import (
	"strings"

	"github.com/RussellLuo/kok/gen/util/generator"
	"github.com/RussellLuo/kok/gen/util/reflector"
)

var (
	template = `// Code generated by kok; DO NOT EDIT.
// github.com/RussellLuo/kok

package {{.Result.PkgName}}

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	{{- range .Result.Imports}}
	"{{.}}"
	{{- end }}

	{{- range .TestSpec.Imports}}
	{{.Alias}} {{.Path}}
	{{- end }}
)

{{- $srcPkgPrefix := .Result.SrcPkgPrefix}}
{{- $interfaceName := .Result.Interface.Name}}
{{- $mockInterfaceName := printf "%s%s" $interfaceName "Mock"}}

// Ensure that {{$mockInterfaceName}} does implement {{$srcPkgPrefix}}{{$interfaceName}}.
var _ {{$srcPkgPrefix}}{{$interfaceName}} = &{{$mockInterfaceName}}{}

type {{$mockInterfaceName}} struct {
{{- range .Result.Interface.Methods}}
	{{.Name}}Func func({{joinParams .Params "$Name $Type" ", "}}) ({{joinParams .Returns "$Name $Type" ", "}})
{{- end}}
}

{{- range .Result.Interface.Methods}}

func (mock *{{$mockInterfaceName}}) {{.Name}}({{joinParams .Params "$Name $Type" ", "}}) ({{joinParams .Returns "$Name $Type" ", "}}) {
	if mock.{{.Name}}Func == nil {
		panic("{{$mockInterfaceName}}.{{.Name}}Func: not implemented")
	}
	return mock.{{.Name}}Func({{joinParams .Params "$CallName" ", "}})
}
{{- end}}

type request struct {
	method string
	path   string
	header map[string]string
	body   string
}

func (r request) ServedBy(handler http.Handler) *httptest.ResponseRecorder {
	var req *http.Request
	if r.body != "" {
		reqBody := strings.NewReader(r.body)
		req = httptest.NewRequest(r.method, r.path, reqBody)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	} else {
		req = httptest.NewRequest(r.method, r.path, nil)
	}

	for key, value := range r.header {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

type response struct {
	statusCode  int
	contentType string
	body        []byte
}

func (want response) Equal(w *httptest.ResponseRecorder) string {
	resp := w.Result()
	gotBody, _ := ioutil.ReadAll(resp.Body)

	gotStatusCode := resp.StatusCode
	if gotStatusCode != want.statusCode {
		return fmt.Sprintf("StatusCode: got (%d), want (%d)", gotStatusCode, want.statusCode)
	}

	wantContentType := want.contentType
	if wantContentType == "" {
		wantContentType = "application/json; charset=utf-8"
	}

	gotContentType := resp.Header.Get("Content-Type")
	if gotContentType != wantContentType {
		return fmt.Sprintf("ContentType: got (%q), want (%q)", gotContentType, wantContentType)
	}

	if reflect.DeepEqual(gotBody, want.body) {
		return fmt.Sprintf("Body: got (%q), want (%q)", gotBody, want.body)
	}

	return ""
}

{{- $codecs := .TestSpec.Codecs}}
{{- range .TestSpec.Tests}}

{{$params := methodParams .Name}}
{{$returns := methodReturns .Name}}
{{$nonCtxParams := nonCtxParams $params}}

func TestHTTP_{{.Name}}(t *testing.T) {
	// in contains all the input parameters (except ctx) of {{.Name}}.
	type in struct {
		{{- range $nonCtxParams}}
		{{.Name}} {{.Type}}
		{{- end}}
	}

	// out contains all the output parameters of {{.Name}}.
	type out struct {
		{{- range $returns}}
		{{.Name}} {{.Type}}
		{{- end}}
	}

	{{if .Cases -}}
	cases := []struct {
		name         string
		request      request
		wantIn       in
		out          out
		wantResponse response
	}{
		{{- range .Cases}}
		{
			name: "{{.Name}}",
			request: request{
				method: "{{.Request.Method}}",
				path:   "{{.Request.Path}}",
				{{- if .Request.Header}}
				header: map[string]string{
					{{- range $key, $value := .Request.Header}}
					"{{$key}}": "{{$value}}",
					{{- end}}
				},
				{{- end}}
				{{- if .Request.Body}}
				body:   ` + "`{{.Request.Body}}`" + `,
				{{- end}}
			},
			wantIn: in{
				{{.WantIn}}
			},
			out: out{
				{{.Out}}
			},
			wantResponse: response{
				statusCode: {{.WantResponse.StatusCode}},
				{{- if .WantResponse.ContentType}}
				contentType: "{{.WantResponse.ContentType}}",
				{{- end}}
				{{- if .WantResponse.Body}}
				body:   []byte({{.WantResponse.Body}}),
				{{- end}}
			},
		},
		{{- end}}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotIn in
			w := c.request.ServedBy(NewHTTPRouter(
				&{{$mockInterfaceName}}{
					{{.Name}}Func: func({{joinParams $params "$Name $Type" ", "}}) ({{joinParams $returns "$Name $Type" ", "}}) {
						gotIn = in{
							{{- range $nonCtxParams}}
							{{.Name}}: {{.Name}},
							{{- end}}
						}
						return {{joinParams $returns "c.out.$Name" ", "}}
					},
				},
				{{$codecs}},
			))

			if !reflect.DeepEqual(gotIn, c.wantIn) {
				t.Fatalf("In: got (%v), want (%v)", gotIn, c.wantIn)
			}

			if errStr := c.wantResponse.Equal(w); errStr != "" {
				t.Fatal(errStr)
			}
		})
	}
	{{- end}}
}
{{- end}}
`
)

type Options struct {
	Formatted bool
}

type Generator struct {
	opts *Options
}

func New(opts *Options) *Generator {
	return &Generator{opts: opts}
}

func (g *Generator) Generate(result *reflector.Result, testFilename string) ([]byte, error) {
	testSpec, err := getTestSpec(testFilename)
	if err != nil {
		return nil, err
	}

	data := struct {
		Result   *reflector.Result
		TestSpec *TestSpec
	}{
		Result:   result,
		TestSpec: testSpec,
	}

	methodMap := make(map[string]*reflector.Method)
	for _, method := range result.Interface.Methods {
		methodMap[method.Name] = method
	}

	return generator.Generate(template, data, generator.Options{
		Funcs: map[string]interface{}{
			"joinParams": func(params []*reflector.Param, format, sep string) string {
				var results []string

				for _, p := range params {
					r := strings.NewReplacer("$Name", p.Name, "$CallName", p.CallName(), "$Type", p.TypeString())
					results = append(results, r.Replace(format))
				}
				return strings.Join(results, sep)
			},
			"methodParams": func(name string) []*reflector.Param {
				method, ok := methodMap[name]
				if !ok {
					return nil
				}
				return method.Params
			},
			"methodReturns": func(name string) []*reflector.Param {
				method, ok := methodMap[name]
				if !ok {
					return nil
				}
				return method.Returns
			},
			"nonCtxParams": func(params []*reflector.Param) (out []*reflector.Param) {
				for _, p := range params {
					if p.Type != "context.Context" {
						out = append(out, p)
					}
				}
				return
			},
		},
		Formatted: g.opts.Formatted,
	})
}
