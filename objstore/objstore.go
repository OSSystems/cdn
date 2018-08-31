package objstore

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OSSystems/cdn/journal"
	"github.com/OSSystems/cdn/pkg/encodedtime"
	"github.com/OSSystems/cdn/storage"

	log "github.com/sirupsen/logrus"
)

var ErrNotFound = errors.New("not found")

type ObjStore struct {
	backend string
	journal *journal.Journal
	storage *storage.Storage
}

func NewObjStore(backend string, journal *journal.Journal, storage *storage.Storage) *ObjStore {
	return &ObjStore{
		backend: backend,
		journal: journal,
		storage: storage,
	}
}

func (obj *ObjStore) Fetch(url string) (*journal.FileMeta, error) {
	log.WithFields(log.Fields{
		"url":     url,
		"backend": obj.backend,
	}).Debug("Fetch file from backend")

	cli := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", obj.backend, url), nil)
	if err != nil {
		return nil, err
	}

	res, err := cli.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ErrNotFound
	}

	filename := obj.FileName(url)

	size, err := obj.storage.Write(filename, res.Body)
	if err != nil {
		return nil, err
	}

	meta := &journal.FileMeta{Name: filename, Hits: 0, Size: size, Timestamp: encodedtime.NewUnix(0)}

	err = obj.journal.AddFile(meta)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"filename": filename,
	}).Debug("File added to objstore")

	return meta, nil
}

func (obj *ObjStore) Get(url string) *journal.FileMeta {
	filename := obj.FileName(url)

	meta, err := obj.journal.Get(filename)
	if err != nil {
		return nil
	}

	_, err = obj.storage.Read(filename)
	if err != nil {
		log.WithFields(log.Fields{"filename": filename, "err": err}).Error("Failed to read file from storage")
		return nil
	}

	return meta
}

func (obj *ObjStore) Serve(url string) (*journal.FileMeta, *os.File, error) {
	filename := obj.FileName(url)

	log.WithFields(log.Fields{
		"filename": filename,
	}).Info("Serve file from objstore")

	meta := obj.Get(filename)
	if meta == nil {
		log.WithFields(log.Fields{"filename": filename}).Debug("File not found in objstore")

		var err error
		meta, err = obj.Fetch(filename)
		if err != nil {
			log.WithFields(log.Fields{"filename": filename, "err": err}).Warn("Failed to fetch file")
			return nil, nil, ErrNotFound
		}
	}

	f, err := obj.storage.Read(meta.Name)
	if err != nil {
		return meta, nil, ErrNotFound
	}

	return meta, f, nil
}

func (obj *ObjStore) FileName(url string) string {
	return filepath.Base(url)
}
