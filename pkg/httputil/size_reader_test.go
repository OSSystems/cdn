package httputil

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSizeReader(t *testing.T) {
	data := make([]byte, 8)
	rand.Read(data)

	rd := NewSizeReader(bytes.NewReader(data), uint64(len(data)), time.Millisecond)

	buf := make([]byte, 4)

	n, err := rd.Read(buf)
	assert.Equal(t, len(buf), n)
	assert.NoError(t, err)

	n, err = rd.Read(buf)
	assert.Equal(t, len(buf), n)
	assert.NoError(t, err)

	n, err = rd.Read(buf)
	assert.Equal(t, 0, n)
	assert.EqualError(t, err, io.EOF.Error())
}
