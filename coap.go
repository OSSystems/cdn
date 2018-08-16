package main

import (
	"net"
	"os"

	coap "github.com/dustin/go-coap"
)

type coapHandler struct {
}

func (h *coapHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	once := lockFile(req.PathString())
	defer once.Do(func() { unlockFile(req.PathString()) })

	if containsFile(req.PathString()) != nil {
		err := fetchFile(req.PathString())
		if err != nil {
			panic(err)
		}
	}

	once.Do(func() { unlockFile(req.PathString()) })

	f, err := os.Open(getFileName(req.PathString()))
	if err != nil {
		panic(err)
	}

	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}

	_, err = f.Seek(int64(req.Block2.Num*req.Block2.Size), 0)
	if err != nil {
		panic(err)
	}

	payload := make([]byte, req.Block2.Size)

	_, err = f.Read(payload)
	if err != nil {
		panic(err)
	}

	msg := &coap.Message{
		Type:      coap.Acknowledgement,
		MessageID: req.MessageID,
		Token:     req.Token,
		Code:      coap.Content,
		Payload:   payload,
	}

	msg.AddOption(coap.Size2, uint32(fi.Size()))

	return msg
}
