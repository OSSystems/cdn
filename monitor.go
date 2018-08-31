package main

import (
	"time"
)

type Monitor interface {
	Init()
	RecordMetric(path string, addr string, transferred int64, size int64, timestamp time.Time)
}

type dummyMonitor struct {
}

func (d *dummyMonitor) Init() {
}

func (d *dummyMonitor) RecordMetric(path string, addr string, transferred int64, size int64, timestamp time.Time) {
}
