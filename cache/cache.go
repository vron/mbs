package cache

import (
	"bytes"
	"time"

	"github.com/boltdb/bolt"
)

// TODO: implement deletion of old no longer useful values...
// TODO: in order to delete files that no longer exist we must understand the globbing pattern used...

const ValueSize = 28

func Open(path string) (*Cache, error) {
	c := Cache{
		path: path,
	}
	return c.open()
}

var bkt = []byte("files")

type Cache struct {
	db   *bolt.DB
	path string
	err  error
}

// Set sets the key to value, returning false if the last call to Set for this
// key provided an identical value, true otherwise. Note in particular that if
// an underlying error occures true will be returned.
func (c *Cache) Set(key string, value []byte) (changed bool) {
	if len(value) != ValueSize {
		// check done to enable future optimization with fixed sized records.
		panic("value with bad length provided")
	}

	// do as much work as possible outside a transaction
	k := []byte(key)
	changed = true

	err := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bkt)
		cv := b.Get(k)
		if bytes.Equal(value, cv) {
			changed = false
		}
		return b.Put(k, value)
	})
	if err != nil {
		changed = true
		if c.err == nil {
			c.err = err
		}
	}
	return
}

func (c *Cache) Err() error {
	return c.err
}
func (c *Cache) Close() error {
	return c.db.Close()
}

func (c *Cache) open() (*Cache, error) {
	db, err := bolt.Open(c.path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	// TODO: Check for file type version etc? (or in file name?)
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bkt)
		return err
	}); err != nil {
		return nil, err
	}
	c.db = db
	return c, nil
}
