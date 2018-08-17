package main

import (
	"net"
	"os"
	"time"

	coap "github.com/OSSystems/go-coap"
)

type coapHandler struct {
}

func (h *coapHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	once := lockFile(req.PathString())
	defer once.Do(func() { unlockFile(req.PathString()) })

	msg := &coap.Message{
		Type:      coap.Acknowledgement,
		MessageID: req.MessageID,
		Token:     req.Token,
	}

	if containsFile(req.PathString()) != nil {
		err := fetchFile(req.PathString())
		if err != nil {
			msg.Code = coap.InternalServerError
			return msg
		}
	}

	once.Do(func() { unlockFile(req.PathString()) })

	f, err := os.Open(getFileName(req.PathString()))
	if err != nil {
		msg.Code = coap.NotFound
		return msg
	}

	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		msg.Code = coap.InternalServerError
		return msg
	}

	_, err = f.Seek(int64(req.Block2.Num*req.Block2.Size), 0)
	if err != nil {
		msg.Code = coap.InternalServerError
		return msg
	}

	payload := make([]byte, req.Block2.Size)

	n, err := f.Read(payload)
	if err != nil {
		msg.Code = coap.InternalServerError
		return msg
	}

	msg.Code = coap.Content
	msg.Payload = payload[0:n]

	msg.AddOption(coap.Size2, uint32(fi.Size()))

	if logger != nil {
		logger.Log(req.PathString(), a.String(), n, fi.Size(), time.Now())
	}

	return msg
}
