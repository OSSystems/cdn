package main

import (
	"context"
	"crypto/tls"
	"errors"
	"math"
	"net"
	"os"
	"sync"
	"time"

	coap "github.com/OSSystems/go-coap"
	"github.com/pion/dtls/v4"
	"github.com/pion/dtls/v4/pkg/crypto/ciphersuite"
	"github.com/pion/dtls/v4/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

const (
	coapsMaxDatagram     = 1500
	coapsHandshakeTimout = 30 * time.Second
	coapsIdleTimeout     = 2 * time.Minute
)

// certificateFile reloads the key pair when either file changes, so a rotated
// Kubernetes Secret takes effect on the next handshake without a restart.
type certificateFile struct {
	certPath, keyPath string

	mu      sync.Mutex
	cert    *tls.Certificate
	modTime time.Time
}

func newCertificateFile(certPath, keyPath string) (*certificateFile, error) {
	c := &certificateFile{certPath: certPath, keyPath: keyPath}
	if _, err := c.get(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *certificateFile) get() (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	latest, err := latestModTime(c.certPath, c.keyPath)
	if err != nil {
		if c.cert != nil {
			return c.cert, nil
		}
		return nil, err
	}

	if c.cert != nil && !latest.After(c.modTime) {
		return c.cert, nil
	}

	cert, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
	if err != nil {
		if c.cert != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Keeping the previous CoAPS certificate")
			return c.cert, nil
		}
		return nil, err
	}

	if c.cert != nil {
		log.Info("Reloaded CoAPS certificate")
	}

	c.cert, c.modTime = &cert, latest

	return c.cert, nil
}

func latestModTime(paths ...string) (time.Time, error) {
	var latest time.Time
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return time.Time{}, err
		}
		if fi.ModTime().After(latest) {
			latest = fi.ModTime()
		}
	}

	return latest, nil
}

func listenCOAPS(addr string, cert *certificateFile) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	return dtls.ListenAddr("udp", udpAddr,
		dtls.WithGetCertificate(func(*dtls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert.get()
		}),
		dtls.WithMinVersion(protocol.Version1_2),
		dtls.WithCipherSuites(
			ciphersuite.TLS_AES_128_GCM_SHA256,
			ciphersuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			ciphersuite.TLS_ECDHE_ECDSA_WITH_AES_128_CCM,
			ciphersuite.TLS_ECDHE_ECDSA_WITH_AES_128_CCM_8,
		),
		dtls.WithExtendedMasterSecret(dtls.RequestExtendedMasterSecret),
	)
}

func (app *App) serveCOAPS(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return err
			}
			log.WithFields(log.Fields{"err": err}).Debug("CoAPS accept failed")
			continue
		}

		go app.serveCOAPSConn(conn)
	}
}

func (app *App) serveCOAPSConn(conn net.Conn) {
	defer conn.Close()

	if dc, ok := conn.(*dtls.Conn); ok {
		ctx, cancel := context.WithTimeout(context.Background(), coapsHandshakeTimout)
		err := dc.HandshakeContext(ctx)
		cancel()
		if err != nil {
			log.WithFields(log.Fields{"addr": conn.RemoteAddr(), "err": err}).Debug("CoAPS handshake failed")
			return
		}
	}

	buf := make([]byte, coapsMaxDatagram)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(coapsIdleTimeout)); err != nil {
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		req, err := coap.ParseMessage(buf[:n])
		if err != nil {
			log.WithFields(log.Fields{"addr": conn.RemoteAddr(), "err": err}).Debug("Invalid CoAPS message")
			continue
		}

		addr, _ := conn.RemoteAddr().(*net.UDPAddr)
		resp := app.handleCOAP(addr, &req)
		if resp == nil {
			continue
		}

		data, err := resp.MarshalBinary()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Failed to encode CoAPS response")
			continue
		}

		if _, err := conn.Write(data); err != nil {
			return
		}
	}
}

// handleCOAP applies the block-wise defaults that go-coap's UDP server applies
// around ServeCOAP, so CoAPS clients get identical responses.
func (app *App) handleCOAP(addr *net.UDPAddr, req *coap.Message) *coap.Message {
	if req.Block2 == nil {
		req.Block2 = &coap.Block{Num: 0, More: false, Size: 1024}
	}

	resp := app.ServeCOAP(nil, addr, req)
	if resp == nil {
		return nil
	}

	header, _ := resp.MarshalBinary()
	size2, ok := resp.Option(coap.Size2).(uint32)
	if !ok || len(header)+int(size2)+1 <= coapsMaxDatagram {
		return resp
	}

	count := math.Floor(float64(size2) / float64(req.Block2.Size))
	resp.Block2 = &coap.Block{
		Num:  req.Block2.Num,
		More: float64(req.Block2.Num) < count,
		Size: req.Block2.Size,
	}
	resp.AddOption(coap.Block2, resp.Block2.MarshalBinary())

	return resp
}
