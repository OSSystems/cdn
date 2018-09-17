package main

import (
	"net/http"

	"github.com/labstack/echo"
)

type Stats struct {
	Size  int64 `json:"size"`
	Count int   `json:"count"`
}

func (app *App) handleStats(c echo.Context) error {
	stats := &Stats{
		Size:  app.journal.Size(),
		Count: app.journal.Count(),
	}
	return c.JSON(http.StatusOK, stats)
}

func (app *App) handlePurge(c echo.Context) error {
	for i := app.journal.Count(); i >= 0; i-- {
		list, err := app.journal.LeastPopular()
		if err != nil {
			continue
		}

		if len(list) > 0 {
			err = app.journal.Delete(list[0])
		} else {
			break
		}
	}

	return nil
}

func (app *App) handleQuery(c echo.Context) error {
	meta, err := app.journal.Get(c.QueryParam("uri"))
	if err != nil {
		return echo.NotFoundHandler(c)
	}

	return c.JSON(http.StatusOK, meta)
}
