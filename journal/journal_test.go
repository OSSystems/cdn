package journal

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
)

func TestNewJournal(t *testing.T) {
	dbFile, err := ioutil.TempFile("", "journal")
	defer os.Remove(dbFile.Name())
	assert.NoError(t, err)

	db, err := bolt.Open(dbFile.Name(), 0600, nil)
	assert.NoError(t, err)

	j := NewJournal(db, 0)
	assert.NotNil(t, j)
}

func TestJournalBasicOperations(t *testing.T) {
	dbFile, err := ioutil.TempFile("", "journal")
	defer os.Remove(dbFile.Name())
	assert.NoError(t, err)

	db, err := bolt.Open(dbFile.Name(), 0600, nil)
	assert.NoError(t, err)

	j := NewJournal(db, 1)
	assert.NotNil(t, j)

	f1 := &FileMeta{Name: "test.txt", Size: 1, Hits: 0}

	err = j.Put(f1)
	assert.NoError(t, err, "Failed to put file metadata")

	f2, err := j.Get(f1.Name)
	assert.NoError(t, err, "Failed to get file metadata")

	if diff := deep.Equal(f1, f2); diff != nil {
		t.Errorf("Getted file metadata is different from original file metadata: %s", diff)
	}

	err = j.Delete(f1)
	assert.NoError(t, err, "Failed to delete file metadata")

	err = j.Hit(f1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), f1.Hits)
}

func TestJournalAddFile(t *testing.T) {
	dbFile, err := ioutil.TempFile("", "journal")
	defer os.Remove(dbFile.Name())
	assert.NoError(t, err)

	db, err := bolt.Open(dbFile.Name(), 0600, nil)
	assert.NoError(t, err)

	j := NewJournal(db, 10)
	assert.NotNil(t, j)

	f1 := &FileMeta{Name: "f1", Size: 5, Hits: 1}
	f2 := &FileMeta{Name: "f2", Size: 5, Hits: 2}
	f3 := &FileMeta{Name: "f3", Size: 5, Hits: 3}

	err = j.AddFile(f1)
	assert.NoError(t, err)
	err = j.AddFile(f2)
	assert.NoError(t, err)

	err = j.Put(f3)
	assert.EqualError(t, err, ErrNotEnoughSpace.Error())

	err = j.AddFile(f3)
	assert.NoError(t, err)

	assert.Equal(t, 2, j.Count())

	f1, err = j.Get(f1.Name)
	assert.Nil(t, f1, "f1 should be removed because it is latest unpopular file metadata")
	assert.Error(t, err)
}

func TestJournalSizeCount(t *testing.T) {
	dbFile, err := ioutil.TempFile("", "journal")
	defer os.Remove(dbFile.Name())
	assert.NoError(t, err)

	db, err := bolt.Open(dbFile.Name(), 0600, nil)
	assert.NoError(t, err)

	j := NewJournal(db, 10)
	assert.NotNil(t, j)

	f1 := &FileMeta{Name: "f1", Size: 5, Hits: 0}
	f2 := &FileMeta{Name: "f2", Size: 5, Hits: 0}

	err = j.AddFile(f1)
	assert.NoError(t, err)
	err = j.AddFile(f2)
	assert.NoError(t, err)

	assert.Equal(t, int64(10), j.Size())
	assert.Equal(t, 2, j.Count())
}
