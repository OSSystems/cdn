package main

import "time"

type Logger interface {
	Init()
	Log(path string, addr string, bytes int, size int64, timestamp time.Time)
}
