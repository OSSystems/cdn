package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OSSystems/cdn/journal"
	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/cdn/storage"
	"github.com/boltdb/bolt"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestInternalHandler(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	app := &App{
		journal: journal.NewJournal(db, -1),
		storage: storage.NewStorage(dir),
		monitor: &dummyMonitor{},
	}

	data := make([]byte, 4)
	rand.Read(data)

	sv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))

	sv.Start()
	defer sv.Close()

	app.objstore = objstore.NewObjStore(fmt.Sprintf("http://%s", sv.Listener.Addr().String()), app.journal, app.storage)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/file", nil)
	req.Header.Add("X-Backend", fmt.Sprintf("http://%s", sv.Listener.Addr().String()))

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, app.internalHandler(c)) {
		assert.Equal(t, http.StatusOK, c.Response().Status)

		body, err := ioutil.ReadAll(rec.Body)
		assert.NoError(t, err)
		assert.Equal(t, data, body)
	}
}
