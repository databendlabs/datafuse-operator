package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// PathToOSFile gets the absolute path from relative path.
func PathToOSFile(relativePath string) (*os.File, error) {
	path, err := filepath.Abs(relativePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed generate absolute file path of %s", relativePath))
	}

	manifest, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to open file %s", path))
	}

	return manifest, nil
}
