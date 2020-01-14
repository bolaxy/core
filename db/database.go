package db

const IdealBatchSize = 25

type Sinker interface {
	Put(key, val []byte) error
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	NewIterator(reverse bool) Iterator
	NewBatch() Batch
	Close() error
	DBPath() string
}

type Iterator interface {
	Item() Item
	Valid() bool
	ValidForPrefix(prefix []byte) bool
	Close()
	Next()
	Seek(key []byte)
	Rewind()
}

type Item interface {
	Key() []byte
	Value() ([]byte, error)
}

type Batch interface {
	Set(key, value []byte) error
	Delete(key []byte) error
	Commit() error
	Cancel()
	SetMaxPendingTxns(max int)
}

// Putter wraps the database write operation supported by both batches and regular databases.
type Putter interface {
	Put(key []byte, value []byte) error
}

// Deleter wraps the database delete operation supported by both batches and regular databases.
type Deleter interface {
	Delete(key []byte) error
}