package main

import (
	"net/http"
	"time"

	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/pkg/httputil"
	"github.com/labstack/echo"
)

func (app *App) handleHTTP(c echo.Context) error {
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
	http.ServeContent(wc, c.Request(), meta.Name, time.Time(meta.Timestamp), httputil.NewSizeReader(f, uint64(meta.Size), time.Second*10))

	if c.Response().Status == http.StatusOK {
		app.monitor.RecordMetric("http", c.Request().URL.String(), c.Request().RemoteAddr, int64(wc.Count()), meta.Size, time.Now())
	}

	return nil
}
