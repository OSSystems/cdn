package main

import (
	"net"
	"time"

	coap "github.com/OSSystems/go-coap"
)

type coapHandler struct {
}

func (h *coapHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	path := req.PathString()

	msg := &coap.Message{
		Type:      coap.Acknowledgement,
		MessageID: req.MessageID,
		Token:     req.Token,
	}

	meta := app.objstore.Contains(path)
	if meta == nil {
		var err error
		meta, err = app.objstore.Fetch(path)
		if err != nil {
			msg.Code = coap.InternalServerError
			return msg
		}
	}

	f, err := app.storage.Read(meta.Name)
	if err != nil {
		msg.Code = coap.NotFound
		return msg
	}

	defer f.Close()

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

	msg.AddOption(coap.Size2, uint32(meta.Size))

	// is the last block?
	if int64(req.Block2.Num*req.Block2.Size) >= meta.Size-int64(req.Block2.Size) {
		app.journal.Hit(meta)
	}

	app.monitor.RecordMetric(req.PathString(), a.String(), n, meta.Size, time.Now())

	return msg
}
