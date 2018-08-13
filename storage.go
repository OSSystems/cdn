package main

import (
	"os"
	"path/filepath"
)

func containsFile(path string) error {
	_, err := os.Stat(getFileName(path))
	return err
}

func fetchFile(url string) error {
	return nil
}

func getFileName(path string) string {
	return filepath.Base(path)
}
