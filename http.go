package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo"
)

func handleHTTP(c echo.Context) error {
	path := c.Request().URL.Path[1:]

	meta := app.objstore.Contains(path)
	if meta == nil {
		var err error
		meta, err = app.objstore.Fetch(path)
		if err != nil {
			return err
		}
	}

	err := app.journal.Hit(meta)
	if err != nil {
		return err
	}

	f, err := app.storage.Read(meta.Name)
	if err != nil {
		return err
	}

	defer f.Close()

	http.ServeContent(c.Response(), c.Request(), meta.Name, time.Time(meta.Timestamp), f)

	return nil
}

func logHTTPRequest(c echo.Context, reqBody, resBody []byte) {
	if c.Response().Status == http.StatusOK {
		if app.monitor != nil {
			app.monitor.RecordMetric(c.Request().URL.String(), c.Request().RemoteAddr, len(resBody), int64(len(resBody)), time.Now())
		}
	}
}
