package main

import (
	"fmt"
	"time"
)

type Method int

const (
	ProxyType = iota
	CacheType
)

type monitor struct{}

func (l monitor) Init() {
}

func (l monitor) RecordMetric(protocol, path, addr string, transferred, size int64, timestamp time.Time, transferredMethod Method) {
	fmt.Println(protocol, path, addr, transferred, size, timestamp, transferredMethod)
}

var Monitor monitor
