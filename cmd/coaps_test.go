package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OSSystems/cdn/internal/cluster"
	"github.com/OSSystems/cdn/internal/journal"
	"github.com/OSSystems/cdn/internal/objstore"
	"github.com/OSSystems/cdn/internal/storage"
	"github.com/OSSystems/cdn/pkg/monitors"
	coap "github.com/OSSystems/go-coap"
	"github.com/boltdb/bolt"
	"github.com/pion/dtls/v4"
	"github.com/pion/dtls/v4/pkg/crypto/ciphersuite"
	"github.com/pion/dtls/v4/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T) *testCA {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &testCA{cert: cert, key: key}
}

func (ca *testCA) pool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca.cert)
	return p
}

func (ca *testCA) writeLeaf(t *testing.T, dir, name string, serial int64) (certPath, keyPath string) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certPath = filepath.Join(dir, name+".pem")
	keyPath = filepath.Join(dir, name+".key")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600))

	return certPath, keyPath
}

func startCOAPS(t *testing.T, data []byte) (addr *net.UDPAddr, ca *testCA) {
	dir := t.TempDir()

	db, err := bolt.Open(filepath.Join(dir, "db"), 0600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file", time.Now(), bytes.NewReader(data))
	}))
	t.Cleanup(backend.Close)

	app := &App{
		journal: journal.NewJournal(db, -1),
		storage: storage.NewStorage(dir),
		monitor: &monitors.DummyMonitor{},
		cluster: cluster.NewCluster(),
		cache:   ".*",
	}
	app.objstore = objstore.NewObjStore(fmt.Sprintf("http://%s", backend.Listener.Addr()), app.journal, app.storage)

	ca = newTestCA(t)
	certPath, keyPath := ca.writeLeaf(t, dir, "server", 2)
	cert, err := newCertificateFile(certPath, keyPath)
	require.NoError(t, err)

	l, err := listenCOAPS("127.0.0.1:0", cert)
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	go app.serveCOAPS(l)

	return l.Addr().(*net.UDPAddr), ca
}

func dialCOAPS(addr *net.UDPAddr, roots *x509.CertPool, serverName string) (*dtls.Conn, error) {
	conn, err := dtls.Dial("udp", addr,
		dtls.WithRootCAs(roots),
		dtls.WithServerName(serverName),
		dtls.WithMaxVersion(protocol.Version1_2),
		dtls.WithCipherSuites(ciphersuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func TestCOAPSBlockwiseDownload(t *testing.T) {
	data := make([]byte, 2500)
	rand.Read(data)

	addr, ca := startCOAPS(t, data)

	conn, err := dialCOAPS(addr, ca.pool(), "localhost")
	require.NoError(t, err)
	defer conn.Close()

	var got []byte
	buf := make([]byte, coapsMaxDatagram)
	for num := uint32(0); ; num++ {
		req := &coap.Message{Type: coap.Confirmable, Code: coap.GET, MessageID: uint16(num + 1)}
		req.SetPathString("/file")
		req.AddOption(coap.Block2, (&coap.Block{Num: num, Size: 1024}).MarshalBinary())

		raw, err := req.MarshalBinary()
		require.NoError(t, err)
		_, err = conn.Write(raw)
		require.NoError(t, err)

		require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
		n, err := conn.Read(buf)
		require.NoError(t, err)

		resp, err := coap.ParseMessage(buf[:n])
		require.NoError(t, err)
		require.Equal(t, coap.Content, resp.Code)
		require.Equal(t, req.MessageID, resp.MessageID)
		require.NotNil(t, resp.Block2)
		assert.Equal(t, num, resp.Block2.Num)

		got = append(got, resp.Payload...)
		if !resp.Block2.More {
			break
		}
	}

	assert.Equal(t, data, got)
}

func TestCOAPSRejectsUntrustedServer(t *testing.T) {
	addr, ca := startCOAPS(t, []byte("x"))

	_, err := dialCOAPS(addr, newTestCA(t).pool(), "localhost")
	assert.Error(t, err, "client trusting another root")

	_, err = dialCOAPS(addr, ca.pool(), "coap.example.com")
	assert.Error(t, err, "hostname not in the certificate")
}

func TestCertificateFileReload(t *testing.T) {
	dir := t.TempDir()
	ca := newTestCA(t)

	certPath, keyPath := ca.writeLeaf(t, dir, "server", 2)
	c, err := newCertificateFile(certPath, keyPath)
	require.NoError(t, err)

	first, err := c.get()
	require.NoError(t, err)

	newCert, newKey := ca.writeLeaf(t, dir, "rotated", 3)
	require.NoError(t, os.Rename(newCert, certPath))
	require.NoError(t, os.Rename(newKey, keyPath))
	future := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(certPath, future, future))

	second, err := c.get()
	require.NoError(t, err)
	assert.NotEqual(t, first.Certificate[0], second.Certificate[0])

	require.NoError(t, os.WriteFile(keyPath, []byte("garbage"), 0600))
	later := future.Add(time.Minute)
	require.NoError(t, os.Chtimes(keyPath, later, later))

	kept, err := c.get()
	require.NoError(t, err)
	assert.Equal(t, second.Certificate[0], kept.Certificate[0])
}
