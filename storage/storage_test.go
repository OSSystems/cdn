package storage

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStorage(t *testing.T) {
	s := NewStorage("prefix")
	assert.NotNil(t, s)
	assert.Equal(t, "prefix", s.Prefix)
}

func TestStorageReadWrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.NoError(t, err)

	s := NewStorage(dir)

	data := []byte("data")

	buf := bytes.NewBuffer(data)

	var wg sync.WaitGroup
	wg.Add(1)

	n, err := s.Write("file", buf, &wg)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), n)

	f, err := s.Read("file")
	assert.NoError(t, err)

	data2, err := ioutil.ReadAll(f)
	assert.NoError(t, err)

	assert.Equal(t, data, data2)
}
