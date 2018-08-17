package main

type Logger interface {
	Init()
	Log(path string, bytes int)
}
