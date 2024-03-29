package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OSSystems/cdn/internal/cluster"
	"github.com/OSSystems/cdn/internal/journal"
	"github.com/OSSystems/cdn/internal/objstore"
	"github.com/OSSystems/cdn/internal/storage"
	"github.com/OSSystems/cdn/pkg/monitors"
	"github.com/boltdb/bolt"
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

func (m *mockMonitor) RecordMetric(protocol, path, addr string, transferred, size int64, timestamp time.Time, transferredMethod monitors.Method) {
	m.Called(protocol, path, addr, transferred, size, timestamp, transferredMethod)
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
		cluster: cluster.NewCluster(),
	}

	data := []byte("hello world")

	testCaseHttp := []struct {
		method string
		data   []byte
	}{
		{echo.GET, nil},
		{echo.OPTIONS, nil},
		{echo.PUT, data},
		{echo.POST, data},
		{echo.TRACE, nil},
		{echo.DELETE, data},
	}

	sv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))

	sv.Start()
	defer sv.Close()

	mm.On("RecordMetric", "http", "/file", mock.Anything, int64(len(data)), int64(len(data)), mock.Anything, mock.Anything).Return()

	app.objstore = objstore.NewObjStore(fmt.Sprintf("http://%s", sv.Listener.Addr().String()), app.journal, app.storage)

	e := echo.New()
	for _, tc := range testCaseHttp {
		req := httptest.NewRequest(tc.method, "/file", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, app.handleHTTP(c)) {
			assert.Equal(t, http.StatusOK, c.Response().Status)
			assert.Equal(t, http.StatusOK, c.Response().Status)

			body, err := ioutil.ReadAll(rec.Body)
			assert.NoError(t, err)
			assert.Equal(t, data, body)
		}
	}
	mm.AssertExpectations(t)
}
