package objstore

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gustavosbarreto/cdn/journal"
	"github.com/gustavosbarreto/cdn/storage"
)

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

	filename := obj.FileName(url)

	size, err := obj.storage.Write(filename, res.Body)
	if err != nil {
		return nil, err
	}

	meta := &journal.FileMeta{Name: filename, Hits: 0, Size: size}

	err = obj.journal.AddFile(meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (obj *ObjStore) Contains(url string) *journal.FileMeta {
	filename := obj.FileName(url)

	meta, err := obj.journal.Get(filename)
	if err != nil {
		return nil
	}

	_, err = obj.storage.Read(filename)
	if err != nil {
		return nil
	}

	return meta
}

func (obj *ObjStore) FileName(url string) string {
	return filepath.Base(url)
}
