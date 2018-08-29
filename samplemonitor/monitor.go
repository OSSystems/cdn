package main

import (
	"fmt"
	"time"
)

type monitor struct{}

func (l monitor) Init() {
}

func (l monitor) RecordMetric(path string, addr string, bytes int, size int64, timestamp time.Time) {
	fmt.Println(path, addr, bytes, size, timestamp)
}

var Monitor monitor
