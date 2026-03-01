package context

import (
	"os"
	"path/filepath"
)

func DetectFolder() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(wd), nil
}
