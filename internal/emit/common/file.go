package common

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
)

type WriteOptions struct {
	Check bool
}

func WriteFile(p string, data []byte, opt WriteOptions) (wrote bool, err error) {
	formattedData, err := formatCode(p, data)
	if err != nil {
		fmt.Printf("warning: code format failed for %s: %v\n", p, err)
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return false, fmt.Errorf("mkdir: %w", err)
	}

	existing, readErr := os.ReadFile(p)
	if readErr == nil {
		if bytes.Equal(existing, formattedData) {
			// no change
			return false, nil
		}
		if opt.Check {
			return false, fmt.Errorf("check failed: %s differs", p)
		}
	} else if !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read existing: %w", readErr)
	}

	if opt.Check {
		// file doesn't exist or differs, but check mode forbids writing
		return false, fmt.Errorf("check failed: %s would be written/changed", p)
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, formattedData, 0o644); err != nil {
		return false, fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("rename tmp: %w", err)
	}

	return true, nil
}

func formatCode(p string, data []byte) ([]byte, error) {
	ext := path.Ext(p)
	switch ext {
	case ".go":
		formatted, err := format.Source(data)
		if err != nil {
			return data, err
		}
		return formatted, nil
	case ".ts":
		return data, nil
	default:
		return data, nil
	}
}
