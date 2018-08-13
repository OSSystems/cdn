package main

import (
	"io/ioutil"
	"net"

	coap "github.com/dustin/go-coap"
)

type coapHandler struct {
}

func (h *coapHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	if containsFile(req.PathString()) != nil {
		fetchFile(req.PathString())
	}

	payload, err := ioutil.ReadFile(getFileName(req.PathString()))
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
