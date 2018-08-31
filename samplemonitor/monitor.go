package main

import (
	"fmt"
	"time"
)

type monitor struct{}

func (l monitor) Init() {
}

func (l monitor) RecordMetric(path string, addr string, transferred int, size int64, timestamp time.Time) {
	fmt.Println(path, addr, transferred, size, timestamp)
}

var Monitor monitor
