package main

import (
	"fmt"
	"time"
)

type monitor struct{}

func (l monitor) Init() {
}

func (l monitor) RecordMetric(protocol string, path string, addr string, transferred int, size int64, timestamp time.Time) {
	fmt.Println(protocol, path, addr, transferred, size, timestamp)
}

var Monitor monitor
