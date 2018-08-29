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
      --coap string      CoAP listen address (default "0.0.0.0:5683")
      --db string        Database file (default "state.db")
  -h, --help             help for cdn
      --http string      HTTP listen address (default "0.0.0.0:8080")
      --monitor string   Monitor plugin
      --storage string   Storage dir (default "./")
```

Example:

```
./cdn --backend http://localhost:8000 --monitor samplemonitor/samplemonitor.so
```

## Monitor plugin

To create your own monitor plugin you must implement the following `Monitor` interface:

```go
type Monitor interface {
        Init()
        RecordMetric(path string, addr string, bytes int, size int64, timestamp time.Time)
}
```

See [samplemonitor/monitor.go](samplemonitor/monitor.go) for an working example of monitor plugin implementation.

To build the plugin use:

```
$ cd samplemonitor
$ go build -buildmode=plugin
```