package common

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type WriteOptions struct {
	Check bool
}

func WriteFile(path string, data []byte, opt WriteOptions) (wrote bool, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("mkdir: %w", err)
	}

	existing, readErr := os.ReadFile(path)
	if readErr == nil {
		if bytes.Equal(existing, data) {
			// no change
			return false, nil
		}
		if opt.Check {
			return false, fmt.Errorf("check failed: %s differs", path)
		}
	} else if !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read existing: %w", readErr)
	}

	if opt.Check {
		// file doesn't exist or differs, but check mode forbids writing
		return false, fmt.Errorf("check failed: %s would be written/changed", path)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return false, fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("rename tmp: %w", err)
	}

	return true, nil
}
