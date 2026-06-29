package customvalidator

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

func ValidateModelNameValue(p string) error {
	if p == "" {
		return fmt.Errorf("model name is empty")
	}

	if strings.TrimSpace(p) != p {
		return fmt.Errorf("model name has leading or trailing spaces")
	}

	if len(p) > 512 {
		return fmt.Errorf("model name is too long")
	}

	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("model name contains NUL")
	}

	for _, r := range p {
		if unicode.IsControl(r) {
			return fmt.Errorf("model name contains control character")
		}
	}

	if strings.Contains(p, `\`) {
		return fmt.Errorf("model name contains backslash")
	}

	if strings.Contains(p, "/") {
		return fmt.Errorf("model name must be file name, not path")
	}

	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("model name is absolute path")
	}

	if len(p) >= 2 && p[1] == ':' {
		return fmt.Errorf("model name contains windows drive prefix")
	}

	if filepath.Ext(p) != ".pt" {
		return fmt.Errorf("model name must have .pt extension")
	}

	cleaned := path.Clean(p)
	if cleaned != p {
		return fmt.Errorf("model name is not clean path: %q -> %q", p, cleaned)
	}

	for part := range strings.SplitSeq(p, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("model name contains bad path part: %q", part)
		}
	}

	return nil
}
