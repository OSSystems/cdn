package main

import (
	"net/http"
	"time"

	"github.com/gustavosbarreto/cdn/objstore"
	"github.com/gustavosbarreto/cdn/pkg/httputil"
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

	wc := httputil.NewResponseWriterCounter(c.Response())
	http.ServeContent(wc, c.Request(), meta.Name, time.Time(meta.Timestamp), f)

	if c.Response().Status == http.StatusOK {
		app.monitor.RecordMetric(c.Request().URL.String(), c.Request().RemoteAddr, int(wc.Count()), meta.Size, time.Now())
	}

	return nil
}
