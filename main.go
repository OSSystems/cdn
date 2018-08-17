package main

import (
	"net"
	"os"
	"plugin"

	coap "github.com/dustin/go-coap"
	"github.com/labstack/echo"
)

var logger Logger

func main() {
	if len(os.Args) > 1 {
		plug, err := plugin.Open(os.Args[1])
		if err != nil {
			panic(err)
		}

		sym, err := plug.Lookup("Logger")
		if err != nil {
			panic(err)
		}

		var ok bool
		logger, ok = sym.(Logger)
		if !ok {
			panic("unexpected type from module symbol")
		}

		logger.Init()
	}

	udpAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:5683")
	if err != nil {
		panic(err)
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	go func() {
		err = coap.Serve(udpListener, &coapHandler{})
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		e := echo.New()
		e.HideBanner = true

		e.GET("*", handleHTTP)

		err = e.Start(":8080")
		if err != nil {
			panic(err)
		}
	}()

	select {}
}
