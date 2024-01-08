package objstore

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/OSSystems/cdn/internal/cluster"
	"github.com/OSSystems/cdn/internal/journal"
	"github.com/OSSystems/cdn/internal/storage"
	"github.com/OSSystems/cdn/pkg/encodedtime"
	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
)

func TestNewObjStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	j := journal.NewJournal(db, 9999)
	s := storage.NewStorage(dir)

	obj := NewObjStore("http://localhost", j, s)
	assert.NotNil(t, obj)
}

func TestObjStoreFetch(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	j := journal.NewJournal(db, 9999)
	s := storage.NewStorage(dir)

	data := make([]byte, 4)
	rand.Read(data)

	sv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))

	sv.Start()
	defer sv.Close()

	obj := NewObjStore(fmt.Sprintf("http://%s", sv.Listener.Addr().String()), j, s)

	meta, rd, err := obj.Fetch(http.DefaultTransport.(*http.Transport), "", "/file")
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.NotNil(t, rd)
	assert.Equal(t, "file", meta.Name)
	assert.Equal(t, int64(len(data)), meta.Size)
	assert.Equal(t, int64(0), meta.Hits)
}

func TestObjStoreGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	j := journal.NewJournal(db, 9999)
	s := storage.NewStorage(dir)

	obj := NewObjStore("http://localhost", j, s)

	data := make([]byte, 4)
	rand.Read(data)

	encodedtime.NewUnix(0)

	meta := &journal.FileMeta{Name: "file", Size: int64(len(data)), Hits: 0, Timestamp: encodedtime.NewUnix(0)}

	err = j.Put(meta)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	n, err := s.Write("file", bytes.NewReader(data), &wg)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), n)

	meta2 := obj.Get("/file")
	assert.NotNil(t, meta)
	assert.Equal(t, meta, meta2)
}

func TestObjStoreServe(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	assert.NoError(t, err)

	j := journal.NewJournal(db, 9999)
	s := storage.NewStorage(dir)

	data := make([]byte, 4)
	rand.Read(data)

	sv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))

	sv.Start()
	defer sv.Close()

	obj := NewObjStore(fmt.Sprintf("http://%s", sv.Listener.Addr().String()), j, s)

	meta, f, err := obj.Serve("/file", cluster.NewCluster(), "")
	assert.NoError(t, err)
	assert.NotNil(t, meta)
	assert.NotNil(t, f)
	assert.Equal(t, "file", meta.Name)
	assert.Equal(t, int64(len(data)), meta.Size)
	assert.Equal(t, int64(0), meta.Hits)
}
