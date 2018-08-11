package main

import (
	"github.com/labstack/echo"
)

func handleHTTP(c echo.Context) error {
	return c.File(c.Request().URL.Path[1:])
}
