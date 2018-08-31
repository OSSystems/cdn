package main

import (
	"time"
)

type Monitor interface {
	Init()
	RecordMetric(protocol string, path string, addr string, transferred int64, size int64, timestamp time.Time)
}

type dummyMonitor struct {
}

func (d *dummyMonitor) Init() {
}

func (d *dummyMonitor) RecordMetric(protocol, path, addr string, transferred, size int64, timestamp time.Time) {
}
