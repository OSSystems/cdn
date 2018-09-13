package main

import (
	"net/http"
	"time"

	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/pkg/httputil"
	"github.com/labstack/echo"
)

func (app *App) internalHandler(c echo.Context) error {
	path := c.Request().URL.Path[1:]
	backend := c.Request().Header.Get("X-Backend")

	if backend == "" {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	meta, f, err := app.objstore.Serve(path, app.cluster, backend)
	if err == objstore.ErrNotFound {
		return echo.NotFoundHandler(c)
	}

	defer f.Close()

	sr := httputil.NewSizeReader(f, uint64(meta.Size), time.Second*10)

	if backend != "" {
		http.ServeContent(c.Response(), c.Request(), meta.Name, time.Time(meta.Timestamp), sr)
	}

	return nil
}
