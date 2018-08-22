package main

import (
	"net/http"
	"time"

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

func logHTTPRequest(c echo.Context, reqBody, resBody []byte) {
	if c.Response().Status == http.StatusOK {
		if logger != nil {
			logger.Log(c.Request().URL.String(), c.Request().RemoteAddr, len(resBody), int64(len(resBody)), time.Now())
		}
	}
}
