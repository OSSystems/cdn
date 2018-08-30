package httputil

import (
	"bufio"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type ResponseWriterCounter struct {
	http.ResponseWriter
	count   uint64
	started time.Time
	writer  http.ResponseWriter
}

func NewResponseWriterCounter(rw http.ResponseWriter) *ResponseWriterCounter {
	return &ResponseWriterCounter{
		writer:  rw,
		started: time.Now(),
	}
}

func (counter *ResponseWriterCounter) Write(buf []byte) (int, error) {
	n, err := counter.writer.Write(buf)
	atomic.AddUint64(&counter.count, uint64(n))
	return n, err
}

func (counter *ResponseWriterCounter) Header() http.Header {
	return counter.writer.Header()
}

func (counter *ResponseWriterCounter) WriteHeader(statusCode int) {
	counter.writer.WriteHeader(statusCode)
}

func (counter *ResponseWriterCounter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return counter.writer.(http.Hijacker).Hijack()
}

func (counter *ResponseWriterCounter) CloseNotify() <-chan bool {
	if cn, ok := counter.writer.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}

	return nil
}

func (counter *ResponseWriterCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}

func (counter *ResponseWriterCounter) Started() time.Time {
	return counter.started
}
