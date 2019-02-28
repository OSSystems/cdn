package main

import (
	"fmt"
	"time"

	"github.com/OSSystems/cdn/pkg/monitors"
)

type monitor struct{}

func (l monitor) Init() {
}

func (l monitor) RecordMetric(protocol, path, addr string, transferred, size int64, timestamp time.Time, transferredMethod monitors.Method) {
	fmt.Println(protocol, path, addr, transferred, size, timestamp, transferredMethod)
}

var Monitor monitor
