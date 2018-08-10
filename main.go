package main

import (
	"io/ioutil"
	"net"

	coap "github.com/dustin/go-coap"
)

type coapHandler struct {
}

func (h *coapHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	payload, err := ioutil.ReadFile(req.PathString())
	if err != nil {
		panic(err)
	}

	return &coap.Message{
		Type:      coap.Acknowledgement,
		MessageID: req.MessageID,
		Token:     req.Token,
		Code:      coap.Content,
		Payload:   payload,
	}
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:5683")
	if err != nil {
		panic(err)
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	err = coap.Serve(udpListener, &coapHandler{})
	if err != nil {
		panic(err)
	}
}
