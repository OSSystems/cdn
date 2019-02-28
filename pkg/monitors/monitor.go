package monitors

import (
	"time"
)

type Method int

const (
	ProxyType = iota
	CacheType
)

type Monitor interface {
	Init()
	RecordMetric(protocol string, path string, addr string, transferred int64, size int64, timestamp time.Time, transferredMethod Method)
}

type DummyMonitor struct {
}

func (d *DummyMonitor) Init() {
}

func (d *DummyMonitor) RecordMetric(protocol, path, addr string, transferred, size int64, timestamp time.Time, transferredMethod Method) {
}
