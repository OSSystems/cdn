package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/pkg/httputil"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
)

var ErrJustGetBeCached = errors.New("Just GET method can be cached")

func (app *App) handleHTTP(c echo.Context) error {
	path := c.Request().URL.Path[1:]

	r, err := regexp.Compile(app.cache)
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn("Regular Expresssion not valid. It will be ignored!")
	}

	if !r.MatchString(path) || strings.Compare(app.cache, "") == 0 {
		req, err := http.NewRequest(c.Request().Method, fmt.Sprintf("%s/%s", app.objstore.Backend, path), c.Request().Body)
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
		wc := httputil.NewResponseWriterCounter(c.Response())
		sr := httputil.NewSizeReader(buffer, uint64(len(body)), time.Second*10)

		http.ServeContent(wc, c.Request(), "", time.Now(), sr)

		if c.Response().Status == http.StatusOK {
			app.monitor.RecordMetric("http", c.Request().URL.String(), c.Request().RemoteAddr, int64(wc.Count()), int64(len(body)), time.Now(), ProxyType)
		}

		return nil
	}

	if strings.Compare(c.Request().Method, "GET") == 0 {
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
		wc := httputil.NewResponseWriterCounter(c.Response())

		http.ServeContent(wc, c.Request(), meta.Name, time.Time(meta.Timestamp), sr)

		if c.Response().Status == http.StatusOK {
			app.monitor.RecordMetric("http", c.Request().URL.String(), c.Request().RemoteAddr, int64(wc.Count()), meta.Size, time.Now(), CacheType)
		}
		return nil
	}

	return ErrJustGetBeCached
}
