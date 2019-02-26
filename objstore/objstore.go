package objstore

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/OSSystems/cdn/cluster"
	"github.com/OSSystems/cdn/journal"
	"github.com/OSSystems/cdn/pkg/encodedtime"
	"github.com/OSSystems/cdn/storage"

	log "github.com/sirupsen/logrus"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrMissingContentLength = errors.New("content-length is missing")
)

type ObjStore struct {
	Backend string
	journal *journal.Journal
	storage *storage.Storage
}

func NewObjStore(backend string, journal *journal.Journal, storage *storage.Storage) *ObjStore {
	return &ObjStore{
		Backend: backend,
		journal: journal,
		storage: storage,
	}
}

func (obj *ObjStore) Fetch(transport *http.Transport, backend, url string) (*journal.FileMeta, io.ReadCloser, error) {
	if backend == "" {
		backend = obj.Backend
	}

	log.WithFields(log.Fields{
		"url":     url,
		"backend": backend,
	}).Debug("Fetch file from backend")

	cli := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", backend, url), nil)
	if backend != "" {
		req.Header.Add("X-Backend", backend)
	}

	if err != nil {
		return nil, nil, err
	}

	res, err := cli.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, nil, ErrNotFound
	}

	filename := obj.FileName(url)

	size, err := getContentLength(&res.Header)
	if err != nil {
		return nil, nil, err
	}

	meta := &journal.FileMeta{Name: filename, Hits: 0, Size: size, Timestamp: encodedtime.NewUnix(0)}

	return meta, res.Body, nil
}

func (obj *ObjStore) Get(url string) *journal.FileMeta {
	filename := obj.FileName(url)

	meta, err := obj.journal.Get(filename)
	if err != nil {
		return nil
	}

	f, err := obj.storage.Read(filename)
	if err != nil {
		log.WithFields(log.Fields{"filename": filename, "err": err}).Error("Failed to read file from storage")
		return nil
	}

	// no longer needed, so close the fd to avoid "many files opened" error
	f.Close()

	return meta
}

func (obj *ObjStore) Serve(uri string, cluster *cluster.Cluster, backend string) (*journal.FileMeta, *os.File, error) {
	filename := obj.FileName(uri)

	var wg sync.WaitGroup

	meta := obj.Get(filename)
	if meta == nil {
		log.WithFields(log.Fields{"filename": filename}).Debug("File not found in objstore")

		var err error
		var rd io.ReadCloser

		var transport *http.Transport
		if backend == "" || cluster == nil {
			transport = http.DefaultTransport.(*http.Transport)
		} else {
			transport = cluster.Transport()
		}

		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		meta, rd, err = obj.Fetch(transport, backend, uri)
		if err != nil {
			log.WithFields(log.Fields{"filename": filename, "err": err}).Warn("Failed to fetch file")
			return nil, nil, ErrNotFound
		}

		err = obj.journal.AddFile(meta)
		if err != nil {
			return nil, nil, err
		}

		log.WithFields(log.Fields{
			"filename": meta.Name,
		}).Debug("File added to objstore")

		wg.Add(1)

		go func() {
			_, err := obj.storage.Write(filename, rd, &wg)

			// close the fd after writting file to storage
			rd.Close()

			if err != nil {
				log.WithFields(log.Fields{"filename": filename, "err": err}).Error("Failed to write file")
			}
		}()
	}

	// wait for file to be created
	wg.Wait()

	f, err := obj.storage.Read(meta.Name)
	if err != nil {
		return meta, nil, ErrNotFound
	}

	if backend == "" && cluster != nil {
		cluster.Propagate(uri)
	}

	return meta, f, nil
}

func (obj *ObjStore) FileName(url string) string {
	return filepath.Base(url)
}

func getContentLength(header *http.Header) (int64, error) {
	length := header.Get("Content-Length")
	if length == "" {
		return -1, ErrMissingContentLength
	}

	i, err := strconv.ParseInt(length, 10, 64)
	if err != nil {
		return -1, err
	}

	return i, nil
}
