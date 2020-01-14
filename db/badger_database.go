package db

import (
	"github.com/dgraph-io/badger"
)

var ErrKeyNotFound = badger.ErrKeyNotFound

type BadgerDatabase struct {
	db *badger.DB
	fn string
}

//NewBadgerDatabase opens an existing database or creates a new one if nothing is
//found in path.
func NewBadgerDatabase(path string) (*BadgerDatabase, error) {
	opts := badger.DefaultOptions(path).
		WithSyncWrites(false).
		WithTruncate(true)
	handle, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	database := &BadgerDatabase{
		db: handle,
		fn: path,
	}

	return database, nil
}

func (db *BadgerDatabase) Close() error {
	return db.db.Close()
}

func (db *BadgerDatabase) DBPath() string {
	return db.fn
}

func (db *BadgerDatabase) Put(key, val []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (db *BadgerDatabase) Get(key []byte) ([]byte, error) {
	txn := db.db.NewTransaction(false)
	item, err := txn.Get(key)
	if err != nil {
		return nil, err
	}

	return item.ValueCopy(nil)
}

func (db *BadgerDatabase) Has(key []byte) (bool, error) {
	txn := db.db.NewTransaction(false)
	_, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (db *BadgerDatabase) Delete(key []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (db *BadgerDatabase) NewIterator(reverse bool) Iterator {
	txn := db.db.NewTransaction(false)
	itOpts := badger.DefaultIteratorOptions
	itOpts.Reverse = reverse
	it := txn.NewIterator(itOpts)
	return &BadgerIterator{it}
}

func (db *BadgerDatabase) NewBatch() Batch {
	return &BadgerBatch{db.db.NewWriteBatch()}
}

type BadgerIterator struct {
	it *badger.Iterator
}

func (it *BadgerIterator) Item() Item {
	return &item{it.it.Item()}
}

func (it *BadgerIterator) Valid() bool {
	return it.it.Valid()
}

func (it *BadgerIterator) ValidForPrefix(prefix []byte) bool {
	return it.it.ValidForPrefix(prefix)
}

func (it *BadgerIterator) Close() {
	it.it.Close()
}

func (it *BadgerIterator) Next() {
	it.it.Next()
}

func (it *BadgerIterator) Seek(key []byte) {
	it.it.Seek(key)
}

func (it *BadgerIterator) Rewind() {
	it.it.Rewind()
}

type BadgerBatch struct {
	batch *badger.WriteBatch
}

func (batch *BadgerBatch) Set(key, value []byte) error {
	return batch.batch.Set(key, value)
}

func (batch *BadgerBatch) Delete(key []byte) error {
	return batch.batch.Delete(key)
}

func (batch *BadgerBatch) Commit() error {
	return batch.batch.Flush()
}

func (batch *BadgerBatch) Cancel() {
	batch.batch.Cancel()
}

func (batch *BadgerBatch) SetMaxPendingTxns(max int) {
	batch.batch.SetMaxPendingTxns(max)
}

type item struct {
	*badger.Item
}

func (i *item) Value() ([]byte, error) {
	return i.Item.ValueCopy(nil)
}
