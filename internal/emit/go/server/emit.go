package server

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit/common"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

//go:embed templates/*.go.tpl
var tplFS embed.FS

func Emit(spec *ir.Spec, opt EmitOptions) ([]string, error) {
	if opt.Package == "" {
		opt.Package = "server"
	}

	data, err := BuildServerData(spec, opt.Package)
	if err != nil {
		return nil, err
	}

	var files []string

	if fs, err := emitOne("templates/types.go.tpl", filepath.Join(opt.OutDir, "go-server", "types.gen.go"), data, opt.Check, funcMap()); err != nil {
		return nil, err
	} else {
		files = append(files, fs...)
	}

	if fs, err := emitOne("templates/transport.go.tpl", filepath.Join(opt.OutDir, "go-server", "transport.go"), data, opt.Check, funcMap()); err != nil {
		return nil, err
	} else {
		files = append(files, fs...)
	}

	if fs, err := emitOne("templates/server.go.tpl", filepath.Join(opt.OutDir, "go-server", "server.gen.go"), data, opt.Check, funcMap()); err != nil {
		return nil, err
	} else {
		files = append(files, fs...)
	}

	return files, nil
}

func emitOne(tplPath, outPath string, data any, check bool, fm template.FuncMap) ([]string, error) {
	tplText, err := tplFS.ReadFile(tplPath)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", tplPath, err)
	}
	tpl, err := template.New(filepath.Base(tplPath)).Funcs(fm).Parse(string(tplText))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", tplPath, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("exec template %s: %w", tplPath, err)
	}

	wrote, err := common.WriteFile(outPath, buf.Bytes(), common.WriteOptions{Check: check})
	if err != nil {
		return nil, err
	}
	if wrote {
		return []string{outPath}, nil
	}
	return []string{}, nil
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"goMethodName": func(opID string) string {
			return GoPublicIdent(opID)
		},
		"enumConst": func(pkg, typeName, val string) string {
			// const name: <TypeName><Val> (sanitized)
			base := GoPublicIdent(typeName) + GoPublicIdent(val)
			if base == "" {
				base = "EnumValue"
			}
			return base
		},
	}
}
