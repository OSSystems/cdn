package main

import (
	"net"

	coap "github.com/dustin/go-coap"
	"github.com/labstack/echo"
)

func main() {
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

		e.GET("*", handleHTTP)

		err = e.Start(":8080")
		if err != nil {
			panic(err)
		}
	}()

	select {}
}
