package main

import (
	"net/http"
	"time"

	"github.com/gustavosbarreto/cdn/objstore"
	"github.com/labstack/echo"
)

func handleHTTP(c echo.Context) error {
	path := c.Request().URL.Path[1:]

	meta, f, err := app.objstore.Serve(path)
	if err == objstore.ErrNotFound {
		return echo.NotFoundHandler(c)
	}

	defer f.Close()

	err = app.journal.Hit(meta)
	if err != nil {
		return err
	}

	http.ServeContent(c.Response(), c.Request(), meta.Name, time.Time(meta.Timestamp), f)

	return nil
}

func logHTTPRequest(c echo.Context, reqBody, resBody []byte) {
	if c.Response().Status == http.StatusOK {
		app.monitor.RecordMetric(c.Request().URL.String(), c.Request().RemoteAddr, len(resBody), int64(len(resBody)), time.Now())
	}
}
