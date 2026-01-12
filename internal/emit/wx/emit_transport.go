package wx

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit/common"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

//go:embed templates/transport.ts.tpl
var transportTplFS embed.FS

func EmitTransport(spec *ir.Spec, opt EmitOptions) ([]string, error) {
	_ = spec

	tplText, err := transportTplFS.ReadFile("templates/transport.ts.tpl")
	if err != nil {
		return nil, fmt.Errorf("read transport template: %w", err)
	}
	tpl, err := template.New("transport").Parse(string(tplText))
	if err != nil {
		return nil, fmt.Errorf("parse transport template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, struct{}{}); err != nil {
		return nil, fmt.Errorf("exec transport template: %w", err)
	}

	outPath := filepath.Join(opt.OutDir, "transport.ts")
	wrote, err := common.WriteFile(outPath, buf.Bytes(), common.WriteOptions{Check: opt.Check})
	if err != nil {
		return nil, err
	}
	if wrote {
		return []string{outPath}, nil
	}
	return []string{}, nil
}
