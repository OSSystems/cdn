package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/pkg/httputil"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
)

func (app *App) handleHTTPGet(c echo.Context) error {
	path := c.Request().URL.Path[1:]

	r, err := regexp.Compile(app.cache)
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn("Regular Expresssion not valid. It will be ignored!")
	}

	wc := httputil.NewResponseWriterCounter(c.Response())

	if !r.MatchString(path) {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", app.objstore.Backend, path), nil)
		if err != nil {
			return err
		}

		req.Header = c.Request().Header

		cli := &http.Client{}

		res, err := cli.Do(req)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		buffer := bytes.NewReader(body)
		sr := httputil.NewSizeReader(buffer, uint64(len(body)), time.Second*10)

		http.ServeContent(wc, c.Request(), "", time.Now(), sr)

		if c.Response().Status == http.StatusOK {
			app.monitor.RecordMetric("http", c.Request().URL.String(), c.Request().RemoteAddr, int64(wc.Count()), int64(len(body)), time.Now())
		}
		return nil
	}

	meta, f, err := app.objstore.Serve(path, app.cluster, "")
	if err == objstore.ErrNotFound {
		return echo.NotFoundHandler(c)
	}

	defer f.Close()

	err = app.journal.Hit(meta)
	if err != nil {
		return err
	}

	sr := httputil.NewSizeReader(f, uint64(meta.Size), time.Second*10)

	http.ServeContent(wc, c.Request(), meta.Name, time.Time(meta.Timestamp), sr)

	if c.Response().Status == http.StatusOK {
		app.monitor.RecordMetric("http", c.Request().URL.String(), c.Request().RemoteAddr, int64(wc.Count()), meta.Size, time.Now())
	}

	return nil
}
