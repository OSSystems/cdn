package httputil

import (
	"io"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

// SizeReader reads from reader while the number of bytes does not reach the size
type SizeReader struct {
	io.ReadSeeker
	reader  io.ReadSeeker
	count   uint64
	size    uint64
	timeout time.Duration
}

func NewSizeReader(reader io.ReadSeeker, size uint64, timeout time.Duration) *SizeReader {
	return &SizeReader{
		count:   0,
		size:    size,
		reader:  reader,
		timeout: timeout,
	}
}

func (s *SizeReader) Read(b []byte) (int, error) {
	for timeout := time.After(s.timeout); ; {
		select {
		case <-timeout:
			log.Warn("Timeout reading from stream")
			return 0, io.ErrUnexpectedEOF
		default:
		}

		n, err := s.reader.Read(b)
		if n > 0 {
			atomic.AddUint64(&s.count, uint64(n))
		}

		if err != nil && err == io.EOF && s.count < s.size {
			continue // no more bytes available yet, wait for more
		}

		return n, err
	}
}

func (s *SizeReader) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekEnd {
		return int64(s.size), nil
	}

	return s.reader.Seek(offset, whence)
}
