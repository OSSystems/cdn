package main

import "fmt"

type logger struct{}

func (l logger) Init() {
	fmt.Println("Init")
}

func (l logger) Log(path string, bytes int) {
	fmt.Println(path, bytes)
}

var Logger logger
