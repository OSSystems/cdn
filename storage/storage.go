package storage

import (
	"io"
	"os"
	"path"
)

type Storage struct {
	Prefix string
}

func NewStorage(prefix string) *Storage {
	return &Storage{Prefix: prefix}
}

func (s *Storage) Read(filename string) (*os.File, error) {
	return os.Open(path.Join(s.Prefix, filename))
}

func (s *Storage) Write(filename string, rd io.Reader) (int64, error) {
	f, err := os.OpenFile(path.Join(s.Prefix, filename), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	return io.Copy(f, rd)
}
