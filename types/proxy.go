package types

// CommitResponse ...
type CommitResponse struct {
	StateHash                   []byte
	InternalTransactionReceipts []InternalTransactionReceipt
}

// CommitCallback ...
// type CommitCallback func(block Block) (CommitResponse, error)
