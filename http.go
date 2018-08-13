package main

import (
	"github.com/labstack/echo"
)

func handleHTTP(c echo.Context) error {
	url := c.Request().URL.Path[1:]

	once := lockFile(url)
	defer once.Do(func() { unlockFile(url) })

	if containsFile(url) != nil {
		err := fetchFile(url)
		if err != nil {
			return err
		}
	}

	once.Do(func() { unlockFile(url) })

	return c.File(getFileName(url))
}
