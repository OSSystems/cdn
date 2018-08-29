package main

import "time"

type Monitor interface {
	Init()
	RecordMetric(path string, addr string, bytes int, size int64, timestamp time.Time)
}
