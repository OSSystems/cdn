package main

import (
	"github.com/labstack/echo"
)

func handleHTTP(c echo.Context) error {
	url := c.Request().URL.Path[1:]

	if containsFile(url) != nil {
		err := fetchFile(url)
		if err != nil {
			return err
		}
	}

	return c.File(getFileName(url))
}
