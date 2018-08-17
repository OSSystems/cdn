package main

import (
	"fmt"
	"time"
)

type logger struct{}

func (l logger) Init() {
	fmt.Println("Init")
}

func (l logger) Log(path string, addr string, bytes int, size int64, timestamp time.Time) {
	fmt.Println(path, addr, bytes, size, timestamp)
}

var Logger logger
