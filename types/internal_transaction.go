package types

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"

	"github.com/bolaxy/common"
	"github.com/bolaxy/common/hexutil"
	"github.com/bolaxy/config"
	"github.com/bolaxy/crypto"
)

/*******************************************************************************
InternalTransactionBody
*******************************************************************************/

// TransactionType ...
type TransactionType byte

const (
	// PEER_ADD ...
	PEERADD TransactionType = iota
	// PEER_REMOVE ...
	PEERREMOVE

	// PARACHAIN_ADD
	PARACHAINADD
	// PARACHAIN_DEL
	PARACHAINDEL
)

// String ...
func (t TransactionType) String() string {
	switch t {
	case PEERADD:
		return "PEER_ADD"
	case PEERREMOVE:
		return "PEER_REMOVE"
	case PARACHAINADD:
		return "PARACHAIN_ADD"
	case PARACHAINDEL:
		return "PARACHAIN_DEL"
	default:
		return "Unknown TransactionType"
	}
}

// InternalTransactionBody ...
type InternalTransactionBody struct {
	Type TransactionType
	Peer conf.Peer
	Id   common.Address //投票的合约地址
}

//Marshal - json encoding of body
func (i *InternalTransactionBody) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b) //will write to b

	if err := enc.Encode(i); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

//Hash returns the SHA256 hash of the InternalTransactionBody,
func (i *InternalTransactionBody) Hash() ([]byte, error) {
	hashBytes, err := i.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(hashBytes), nil
}

// InternalTransaction ...
type InternalTransaction struct {
	Body      InternalTransactionBody
	Signature string
}

// NewInternalTransaction ...
func NewInternalTransaction(tType TransactionType, peer conf.Peer, id common.Address) InternalTransaction {
	return InternalTransaction{
		Body: InternalTransactionBody{Type: tType, Peer: peer, Id: id},
	}
}

// NewInternalTransactionJoin ...
func NewInternalTransactionJoin(peer conf.Peer) InternalTransaction {
	return NewInternalTransaction(PEERADD, peer, common.Address{})
}

// NewInternalTransactionLeave ...
func NewInternalTransactionLeave(peer conf.Peer) InternalTransaction {
	return NewInternalTransaction(PEERREMOVE, peer, common.Address{})
}

// Marshal ...
func (t *InternalTransaction) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b) //will write to b

	if err := enc.Encode(t); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Unmarshal ...
func (t *InternalTransaction) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	if err := dec.Decode(t); err != nil {
		return err
	}

	return nil
}

//Sign returns the ecdsa signature of the SHA256 hash of the transaction's body
func (t *InternalTransaction) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := t.Body.Hash()
	if err != nil {
		return err
	}

	sig, err := crypto.Sign(signBytes, privKey)
	if err != nil {
		return err
	}

	t.Signature = hexutil.Encode(sig)

	return err
}

// Verify ...
func (t *InternalTransaction) Verify() (bool, error) {
	pubBytes := t.Body.Peer.PubKeyBytes()
	signBytes, err := t.Body.Hash()
	if err != nil {
		return false, err
	}

	sig, err := hexutil.Decode(t.Signature)
	if err != nil {
		return false, err
	}

	return crypto.VerifySignature(pubBytes, signBytes, sig[:len(sig)-1]), nil
}

//HashString returns a string representation of the body's hash. It is used in
//node/core as a key in a map to keep track of InternalTransactions as they are
//being processed asynchronously by the consensus and application.
func (t *InternalTransaction) HashString() string {
	hash, _ := t.Body.Hash()
	return string(hash)
}

//AsAccepted returns a receipt to accept an InternalTransaction
func (t *InternalTransaction) AsAccepted() InternalTransactionReceipt {
	return InternalTransactionReceipt{
		InternalTransaction: *t,
		Accepted:            true,
	}
}

//AsRefused return a receipt to refuse an InternalTransaction
func (t *InternalTransaction) AsRefused() InternalTransactionReceipt {
	return InternalTransactionReceipt{
		InternalTransaction: *t,
		Accepted:            false,
	}
}

/*******************************************************************************
InternalTransactionReceipt
*******************************************************************************/

//InternalTransactionReceipt records the decision by the application to accept
//or refuse and InternalTransaction
type InternalTransactionReceipt struct {
	InternalTransaction InternalTransaction
	Accepted            bool
}
