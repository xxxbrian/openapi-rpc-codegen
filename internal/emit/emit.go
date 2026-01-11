package emit

import (
	"fmt"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	OutDir  string
	Targets []string
	Check   bool
	Verbose bool
}

func Dispatch(spec *ir.Spec, opt Options) ([]string, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil IR spec")
	}
	if len(opt.Targets) == 0 {
		// default target
		opt.Targets = []string{"raw-ir"}
	}
	var files []string
	for _, t := range opt.Targets {
		switch t {
		case "raw-ir":
			// TODO: Emit raw IR for debugging
		default:
			return nil, fmt.Errorf("unknown target: %s", t)
		}
	}
	return files, nil
}
