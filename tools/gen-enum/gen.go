package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/scylladb/go-set/strset"
	"golang.org/x/tools/go/packages"
)

var funcMap = template.FuncMap{
	"join": strings.Join,
}

type GeneratorOptions struct {
	Args         []string
	BuildTags    string
	GenerateFlag bool
	Output       string
	Type         string
	Pointer      bool
}

type Generator struct {
	generateIDs    bool
	options        GeneratorOptions
	pkgDefs        map[*ast.Ident]types.Object
	pkgName        string
	underlyingType string
	values         []Value
}

type Value struct {
	ID           string
	Name         string
	OriginalName string
	Value        int64
	String       string
}

func NewGenerator(options GeneratorOptions) *Generator {
	return &Generator{options: options}
}

func (g *Generator) Run() ([]byte, error) {
	var (
		tags   []string
		output string
	)

	if g.options.BuildTags != "" {
		tags = strings.Split(g.options.BuildTags, ",")
	}

	if g.options.Output == "" {
		output = "."
	}

	cfg := &packages.Config{
		Mode:       packages.LoadSyntax,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}

	pkgs, err := packages.Load(cfg, g.options.Output)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("%d packages found", len(pkgs))
	}

	g.pkgName = pkgs[0].Name
	g.pkgDefs = pkgs[0].TypesInfo.Defs

	pkgFiles := pkgs[0].Syntax

	for _, file := range pkgFiles {
		ast.Inspect(file, g.findType)
	}

	data := struct {
		Args           []string
		GenerateFlag   bool
		GenerateIDs    bool
		PackageName    string
		Pointer        bool
		Type           string
		UnderlyingType string
		Values         []Value
	}{
		Args:           g.options.Args,
		GenerateFlag:   g.options.GenerateFlag,
		GenerateIDs:    g.generateIDs,
		PackageName:    g.pkgName,
		Pointer:        g.options.Pointer,
		Type:           g.options.Type,
		UnderlyingType: g.underlyingType,
		Values:         g.values,
	}

	var buf bytes.Buffer

	if err := _tmpl.Execute(&buf, data); err != nil {
		return buf.Bytes(), err
	}

	output = filepath.Join(output, fmt.Sprintf("%s_enum.go", strings.ToLower(g.options.Type)))

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), err
	}

	if err := os.WriteFile(output, src, 0o644); err != nil {
		return buf.Bytes(), err
	}

	return src, nil
}

func (g *Generator) findType(node ast.Node) bool {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.CONST {
		// Enum declarations need to be const.
		return true
	}

	typ := "" // name of the constant

	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec) // we've already determined this is a const
		if vspec.Type != nil {
			ident, ok := vspec.Type.(*ast.Ident)
			if !ok {
				continue
			}

			typ = ident.Name
		}

		if g.options.Type != typ {
			// Not the type we want.
			continue
		}

		for _, name := range vspec.Names {
			if name.Name == "_" {
				continue // ignore
			}

			obj, ok := g.pkgDefs[name]
			if !ok {
				log.Fatalf("no value for constant %q", typ)
			}

			var isSigned bool

			info := obj.Type().Underlying().(*types.Basic)
			switch info.Kind() {
			case types.Int:
				isSigned = true
			case types.Int8:
				isSigned = true
			case types.Int16:
				isSigned = true
			case types.Int32:
				isSigned = true
			case types.Int64:
				isSigned = true
			case types.Uint:
				isSigned = false
			case types.Uint8:
				isSigned = false
			case types.Uint16:
				isSigned = false
			case types.Uint32:
				isSigned = false
			case types.Uint64:
				isSigned = false
			default:
				log.Fatalf("%q must be an integer type", typ)
			}

			g.underlyingType = info.String()

			value := obj.(*types.Const).Val()
			if value.Kind() != constant.Int {
				log.Fatalf("%q constant is not an integer", name)
			}

			i64, _ := constant.Int64Val(value)
			u64, _ := constant.Uint64Val(value)

			v := Value{
				OriginalName: name.Name,
				Value:        i64,
				String:       value.String(),
			}

			if !isSigned {
				v.Value = int64(u64)
			}

			comment := vspec.Comment
			if comment != nil && len(comment.List) == 1 {
				text := strings.TrimSpace(comment.Text())
				fields := strset.New(strings.Split(text, ", ")...)

				fields.Each(func(field string) bool {
					idx := strings.Index(field, "=")

					if field[:idx] == "name" {
						name := field[idx+1:]

						if name[0] == '"' {
							var err error

							name, err = strconv.Unquote(name)
							if err != nil {
								log.Fatalf("%+v", err)
							}
						}

						v.Name = name
					}

					if strings.Contains(field, "id") {
						v.ID = field[idx+1:]
						g.generateIDs = true
					}

					return true
				})
			} else {
				v.Name = strings.ToLower(strings.TrimSpace(name.Name))
				v.Name = strings.ReplaceAll(v.Name, "_", "-")
			}

			g.values = append(g.values, v)
		}
	}

	return false
}

var _tmpl = template.Must(template.New("").Funcs(funcMap).Parse(`// Code generated by "gen-enum {{ join .Args " " }}"; DO NOT EDIT.
package {{ .PackageName }}

import "errors"

func _() {
	// An "invalid array index" compiler error signifies that the constant
	// values have changed. Run the generator again.
	var x [1]struct{}
        {{- range .Values }}
	_ = x[{{ .OriginalName }}-{{ .Value }}]
        {{- end }}
}

var _{{ .Type }}_string_to_type = map[string]{{ .Type }}{
	{{- range $i, $value := .Values }}
	"{{ $value.Name }}": {{ $value.OriginalName }},
	{{- end }}
}

var _{{ .Type }}_type_to_string = map[{{ .Type }}]string{
	{{- range $i, $value := .Values }}
	{{ $value.OriginalName }}: "{{ $value.Name }}",
	{{- end }}
}

{{ if .GenerateIDs }}
var _{{ .Type }}_id_to_type = map[{{ .UnderlyingType }}]{{ .Type }}{
	{{- range $i, $value := .Values }}
	{{ $value.ID }}: {{ $value.OriginalName }},
	{{- end }}
}

var _{{ .Type }}_type_to_id = map[{{ .Type }}]{{ .UnderlyingType }}{
	{{- range $i, $value := .Values }}
	{{ $value.OriginalName }}: {{ $value.ID }},
	{{- end }}
}
{{ end }}

var ErrInvalid{{ .Type }} = errors.New("invalid {{ .Type }}")

{{ if .GenerateIDs }}
{{ if .Pointer }}
func (i *{{ .Type }}) ID() {{ .UnderlyingType }} {
	return _{{ .Type }}_type_to_id[*i]
}
{{ else }}
func (i {{ .Type }}) ID() {{ .UnderlyingType }} {
	return _{{ .Type }}_type_to_id[i]
}
{{ end }}
{{ end }}

{{ if .Pointer }}
func (i *{{ .Type }}) String() string {
	return _{{ .Type }}_type_to_string[*i]
}
{{ else }}
func (i {{ .Type }}) String() string {
	return _{{ .Type }}_type_to_string[i]
}
{{ end }}

{{ if .GenerateFlag }}
func (i *{{ .Type }}) Set(s string) error {
	if t, ok := _{{ .Type }}_string_to_type[s]; ok {
		*i = t
		return nil
	}
	return ErrInvalid{{ .Type }}
}

func (i *{{ .Type }}) Type() string {
	return i.String()
}
{{ end }}

{{ if .GenerateIDs }}
{{ if .Pointer }}
func IDTo{{ .Type }}(i {{ .UnderlyingType }}) *{{ .Type }} {
	if t, ok := _{{ .Type }}_id_to_type[i]; ok {
		return &t
	}
	return nil
}
{{ else }}
func IDTo{{ .Type }}(i {{ .UnderlyingType }}) {{ .Type }} {
	if t, ok := _{{ .Type }}_id_to_type[i]; ok {
		return t
	}
	return 0
}
{{ end }}
{{ end }}

{{ if .Pointer }}
func StringTo{{ .Type }}(s string) *{{ .Type }} {
	if t, ok := _{{ .Type }}_string_to_type[s]; ok {
		return &t
	}
	return nil
}
{{ else }}
func StringTo{{ .Type }}(s string) {{ .Type }} {
	if t, ok := _{{ .Type }}_string_to_type[s]; ok {
		return t
	}
	return 0
}
{{ end }}

func Is{{ .Type }}(s string) bool {
	if _, ok := _{{ .Type }}_string_to_type[s]; ok {
		return true
	}
	return false
}

func {{ .Type }}List() []{{ .Type }} {
	return []{{ .Type }}{
		{{- range $i, $value := .Values }}
		{{ $value.OriginalName }},
		{{- end }}
	}
}
`))
