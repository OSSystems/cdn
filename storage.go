package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

func containsFile(path string) error {
	_, err := os.Stat(getFileName(path))
	return err
}

func fetchFile(url string) error {
	cli := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8000/%s", url), nil)
	if err != nil {
		return err
	}

	res, err := cli.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	if err = ioutil.WriteFile(getFileName(url), data, 0644); err != nil {
		panic(err)
	}

	return nil
}

func getFileName(path string) string {
	return filepath.Base(path)
}
