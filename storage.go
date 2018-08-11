package main

import "os"

func containsFile(path string) error {
	_, err := os.Stat(path)
	return err
}

func fetchFile(url string) error {
	return nil
}
