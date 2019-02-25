package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/OSSystems/cdn/cluster"
	"github.com/OSSystems/cdn/objstore"
	"github.com/OSSystems/crosscoap"
	coap "github.com/OSSystems/go-coap"
	log "github.com/sirupsen/logrus"
)

const (
	defaultHTTPTimeout = 5 * time.Second
)

func doHTTPRequest(req *http.Request) (*http.Response, []byte, error) {
	timeout := defaultHTTPTimeout
	httpClient := &http.Client{Timeout: timeout}

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer httpResp.Body.Close()
	httpBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}
	return httpResp, httpBody, nil
}

func (app *App) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, req *coap.Message) *coap.Message {
	path := req.PathString()
	r, err := regexp.Compile(app.cache)
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn("Regular Expresssion not valid. It will be ignored!")
	}

	if !(r.MatchString(path)) {
		waitForResponse := req.IsConfirmable()
		request := crosscoap.TranslateCOAPRequestToHTTPRequest(req, app.objstore.Backend)
		if request == nil {
			if waitForResponse {
				return &crosscoap.GenerateBadRequestCOAPResponse(req).Message
			} else {
				req.Code = coap.NotFound
				return req
			}
		}
		responseChan := make(chan *coap.Message, 1)
		go func() {
			httpResp, httpBody, err := doHTTPRequest(request)
			if err != nil {
				log.WithFields(log.Fields{"Erro": err}).Debug("Error on Http request")
			}
			if waitForResponse {
				coapResp, err := crosscoap.TranslateHTTPResponseToCOAPResponse(httpResp, httpBody, err, req)
				if err != nil {
					log.WithFields(log.Fields{"Erro": err}).Debug("Error translating HTTP to CoAP")
				}

				responseChan <- &coapResp.Message
			}
		}()

		if waitForResponse {
			coapResp := <-responseChan

			app.monitor.RecordMetric("coap", req.PathString(), a.String(), int64(len(coapResp.Payload)), int64(0), time.Now())

			return coapResp
		} else {
			req.Code = coap.NotFound
			return req
		}
	}

	msg := &coap.Message{
		Type:      coap.Acknowledgement,
		MessageID: req.MessageID,
		Token:     req.Token,
	}

	var cluster *cluster.Cluster
	if req.Block2.Num == 0 { // only propagate to the cluster on first request
		cluster = app.cluster
	}

	meta, f, err := app.objstore.Serve(path, cluster, "")
	if err == objstore.ErrNotFound {
		msg.Code = coap.NotFound
		return msg
	}

	defer f.Close()

	for timeout := time.After(time.Second * 10); ; {
		select {
		case <-timeout:
			log.Warn("Timeout reading from stream")
			msg.Code = coap.InternalServerError
			return msg
		default:
		}

		fi, err := f.Stat()
		if err != nil {
			msg.Code = coap.InternalServerError
			return msg
		}

		// check if there are bytes available
		if fi.Size() <= int64(req.Block2.Num*req.Block2.Size) {
			continue // no more bytes available yet, wait for more
		}

		_, err = f.Seek(int64(req.Block2.Num*req.Block2.Size), 0)
		if err != nil {
			msg.Code = coap.InternalServerError
			return msg
		}

		break
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

	app.monitor.RecordMetric("coap", req.PathString(), a.String(), int64(n), meta.Size, time.Now())

	return msg
}
