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

	"github.com/boltdb/bolt"
	"github.com/gustavosbarreto/cdn/journal"
	"github.com/gustavosbarreto/cdn/objstore"
	"github.com/gustavosbarreto/cdn/storage"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMonitor struct {
	mock.Mock
}

func (m *mockMonitor) Init() {
	m.Called()
}

func (m *mockMonitor) RecordMetric(path string, addr string, transferred int, size int64, timestamp time.Time) {
	m.Called(path, addr, transferred, size, timestamp)
}

func TestHttpHandler(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	mm := &mockMonitor{}

	app := &App{
		journal: journal.NewJournal(db, -1),
		storage: storage.NewStorage(dir),
		monitor: mm,
	}

	data := make([]byte, 4)
	rand.Read(data)

	sv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))

	sv.Start()
	defer sv.Close()

	mm.On("RecordMetric", "/file", mock.Anything, int64(len(data)), int64(len(data)), mock.Anything).Return()

	app.objstore = objstore.NewObjStore(fmt.Sprintf("http://%s", sv.Listener.Addr().String()), app.journal, app.storage)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/file", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, app.handleHTTP(c)) {
		assert.Equal(t, http.StatusOK, c.Response().Status)
		assert.Equal(t, http.StatusOK, c.Response().Status)

		body, err := ioutil.ReadAll(rec.Body)
		assert.NoError(t, err)
		assert.Equal(t, data, body)
	}

	mm.AssertExpectations(t)
}
