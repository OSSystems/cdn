<p align="center">
    <img align="center" src="docs/logo.png" height="200px"/>
</p>

# cdn

*Multi-protocol Caching Layer for any HTTP backend*

This project aims to provide an caching layer for any HTTP backend, e.g. Amazon S3.

## Usage

```
$ ./cdn --help
Usage:
  cdn [flags]

Flags:
      --backend string   Backend HTTP server URL
      --coap string      CoAP listen address (default "0.0.0.0:5000")
  -h, --help             help for cdn
      --http string      HTTP listen address (default "0.0.0.0:8080")
      --logger string    Logger plugin
```

Example:

```
./cdn --backend http://localhost:8000 --logger printlogger/printlogger.so
```

## Logger plugin

To create your own logger plugin you must implement the following `Logger` interface:

```go
type Logger interface {
        Init()
        Log(path string, addr string, bytes int, size int64, timestamp time.Time)
}
```

See [printlogger/logger.go](printlogger/logger.go) for an working example of logger plugin implementation.

To build the plugin use:

```
$ cd printlogger
$ go build -buildmode=plugin
```