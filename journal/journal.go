package journal

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/gustavosbarreto/cdn/pkg/encodedtime"
)

var journalBucketName = []byte("mapping")

var ErrNotEnoughSpace = errors.New("not enough space")

type Journal struct {
	db      *bolt.DB
	maxSize int64
}

func NewJournal(db *bolt.DB, maxSize int64) *Journal {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(journalBucketName)
		return err
	})
	if err != nil {
		return nil
	}

	return &Journal{db: db, maxSize: maxSize}
}

func (j *Journal) AddFile(meta *FileMeta) error {
	for {
		err := j.Put(meta)
		if err == ErrNotEnoughSpace {
			list, err := j.LeastPopular()
			if err != nil {
				return nil
			}

			if len(list) == 0 {
				return ErrNotEnoughSpace
			}

			err = j.Delete(list[0])
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}

func (j *Journal) Put(meta *FileMeta) error {
	return j.db.Update(func(tx *bolt.Tx) error {
		if j.maxSize > -1 && j.Size()+meta.Size > j.maxSize {
			return ErrNotEnoughSpace
		}

		buf, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		return tx.Bucket(journalBucketName).Put([]byte(meta.Name), buf)
	})
}

func (j *Journal) Get(name string) (*FileMeta, error) {
	fm := new(FileMeta)

	err := j.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(journalBucketName).Get([]byte(name))
		if data == nil {
			return errors.New("error")
		}

		return json.Unmarshal(data, fm)
	})

	if err != nil {
		return nil, err
	}

	return fm, nil
}

func (j *Journal) Delete(fm *FileMeta) error {
	return j.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(journalBucketName).Delete([]byte(fm.Name))
	})
}

func (j *Journal) Hit(fm *FileMeta) error {
	return j.db.Update(func(tx *bolt.Tx) error {
		f := *fm
		f.Hits = f.Hits + 1

		data, err := json.Marshal(f)
		if err != nil {
			return err
		}

		err = tx.Bucket(journalBucketName).Put([]byte(f.Name), data)
		if err != nil {
			return err
		}

		fm.Hits = f.Hits

		return nil
	})
}

func (j *Journal) Size() (size int64) {
	j.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(journalBucketName).Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			f := new(FileMeta)
			err := json.Unmarshal(v, f)
			if err != nil {
				return err
			}

			size = size + f.Size
		}

		return nil
	})

	return
}

func (j *Journal) Count() (count int) {
	j.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(journalBucketName).Stats().KeyN
		return nil
	})

	return
}

func (j *Journal) LeastPopular() ([]*FileMeta, error) {
	var list []*FileMeta

	err := j.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(journalBucketName).Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			f := new(FileMeta)
			err := json.Unmarshal(v, f)
			if err != nil {
				return err
			}

			list = append(list, f)
		}

		sort.Slice(list[:], func(i, j int) bool {
			return list[i].Hits < list[j].Hits
		})

		return nil
	})

	return list, err
}

type FileMeta struct {
	Name      string           `json:"name"`
	Size      int64            `json:"size"`
	Hits      int64            `json:"hits"`
	Timestamp encodedtime.Unix `json:"timestamp"`
}
