package types

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/bolaxy/common/hexutil"
	"github.com/bolaxy/crypto"
)

// EventBody ...
type EventBody struct {
	Transactions         [][]byte              //the payload
	InternalTransactions []InternalTransaction //peers add and removal internal consensus
	Parents              []string              //hashes of the event's parents, self-parent first
	Creator              []byte                //creator's public key
	Index                int                   //index in the sequence of events created by Creator
	BlockSignatures      []BlockSignature      //list of Block signatures signed by the Event's Creator ONLY

	//These fields are not serialized
	CreatorID            uint32
	OtherParentCreatorID uint32
	SelfParentIndex      int
	OtherParentIndex     int
}

//Marshal - json encoding of body only
func (e *EventBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *EventBody) MarshalSign() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	f := &EventBody{
		Transactions:e.Transactions,
		InternalTransactions:e.InternalTransactions,
		Parents:e.Parents,
		Creator :e.Creator,
		Index :e.Index,
		BlockSignatures :e.BlockSignatures,
	}
	if err := enc.Encode(f); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Unmarshal ...
func (e *EventBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(e); err != nil {
		return err
	}
	return nil
}

// Hash ...
func (e *EventBody) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(hashBytes), nil
}

func (e *EventBody) HashSign() ([]byte, error) {
	hashBytes, err := e.MarshalSign()
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(hashBytes), nil
}

// EventCoordinates ...
type EventCoordinates struct {
	Hash  string
	Index int
}

// CoordinatesMap ...
type CoordinatesMap map[string]EventCoordinates

// NewCoordinatesMap ...
func NewCoordinatesMap() CoordinatesMap {
	return make(map[string]EventCoordinates)
}

// Copy ...
func (c CoordinatesMap) Copy() CoordinatesMap {
	res := make(map[string]EventCoordinates, len(c))
	for k, v := range c {
		res[k] = v
	}
	return res
}

// Event ...
type Event struct {
	Body      EventBody
	Signature string //creator's digital signature of body

	TopologicalIndex int

	//used for sorting
	round            *int
	LamportTimestamp *int

	RoundReceived *int

	LastAncestors    CoordinatesMap //[participant pubkey] => last ancestor
	FirstDescendants CoordinatesMap //[participant pubkey] => first descendant

	Creator string
	Hash    []byte
	Hex     string
}

// NewEvent ...
func NewEvent(transactions [][]byte,
	internalTransactions []InternalTransaction,
	blockSignatures []BlockSignature,
	parents []string,
	creator []byte,
	index int) *Event {

	body := EventBody{
		Transactions:         transactions,
		InternalTransactions: internalTransactions,
		BlockSignatures:      blockSignatures,
		Parents:              parents,
		Creator:              creator,
		Index:                index,
	}

	return &Event{
		Body: body,
	}
}

// Creator ...
func (e *Event) GetCreator() string {
	if e.Creator == "" {
		pubKey, _ := crypto.UnmarshalPubkey(e.Body.Creator)
		e.Creator = strings.ToUpper(hexutil.Encode(crypto.CompressPubkey(pubKey)))
	}
	return e.Creator
}

// SelfParent ...
func (e *Event)  SelfParent() string {
	return e.Body.Parents[0]
}

// OtherParent ...
func (e *Event) OtherParent() string {
	return e.Body.Parents[1]
}

// Transactions ...
func (e *Event) Transactions() [][]byte {
	return e.Body.Transactions
}

// InternalTransactions ...
func (e *Event) InternalTransactions() []InternalTransaction {
	return e.Body.InternalTransactions
}

// Index ...
func (e *Event) Index() int {
	return e.Body.Index
}

// BlockSignatures ...
func (e *Event) BlockSignatures() []BlockSignature {
	return e.Body.BlockSignatures
}

//IsLoaded - True if Event contains a payload or is the initial Event of its creator
func (e *Event) IsLoaded() bool {
	if e.Body.Index == 0 {
		return true
	}

	hasTransactions := e.Body.Transactions != nil && len(e.Body.Transactions) > 0
	hasInternalTransactions := e.Body.InternalTransactions != nil && len(e.Body.InternalTransactions) > 0

	return hasTransactions || hasInternalTransactions
}

//Sign signs with an ecdsa sig
func (e *Event) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := e.Body.HashSign()
	if err != nil {
		return err
	}

	sig, err := crypto.Sign(signBytes, privKey)
	if err != nil {
		return err
	}

	e.Signature = hexutil.Encode(sig)

	sig, err = hexutil.Decode(e.Signature)
	if err != nil {
		return err
	}

	return err
}

// Verify ...
func (e *Event) Verify() (bool, error) {

	//first check signatures on internal transactions
	for _, itx := range e.Body.InternalTransactions {
		ok, err := itx.Verify()

		if err != nil {
			return false, err
		} else if !ok {
			return false, fmt.Errorf("invalid signature on internal transaction")
		}
	}

	//then check event signature
	pubBytes := e.Body.Creator
	signBytes, err := e.Body.HashSign()
	if err != nil {
		return false, err
	}

	sig, err := hexutil.Decode(e.Signature)
	if err != nil {
		return false, err
	}
	return crypto.VerifySignature(pubBytes, signBytes, sig[:len(sig)-1]), nil
}

//Marshal - json encoding of body and signature
func (e *Event) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)

	if err := enc.Encode(e); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Unmarshal ...
func (e *Event) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	return dec.Decode(e)
}

//Hash returns sha256 hash of body
func (e *Event) GetHash() ([]byte, error) {
	if len(e.Hash) == 0 {
		hash, err := e.Body.Hash()
		if err != nil {
			return nil, err
		}

		e.Hash = hash
	}

	return e.Hash, nil
}

// Hex ...
func (e *Event) GetHex() string {
	if e.Hex == "" {
		hash, _ := e.GetHash()
		e.Hex = hexutil.Encode(hash)
	}

	return e.Hex
}

// SetRound ...
func (e *Event) SetRound(r int) {
	if e.round == nil {
		e.round = new(int)
	}

	*e.round = r
}

// GetRound ...
func (e *Event) GetRound() *int {
	return e.round
}

// SetLamportTimestamp ...
func (e *Event) SetLamportTimestamp(t int) {
	if e.LamportTimestamp == nil {
		e.LamportTimestamp = new(int)
	}

	*e.LamportTimestamp = t
}

// SetRoundReceived ...
func (e *Event) SetRoundReceived(rr int) {
	if e.RoundReceived == nil {
		e.RoundReceived = new(int)
	}

	*e.RoundReceived = rr
}

// SetWireInfo ...
func (e *Event) SetWireInfo(selfParentIndex int,
	otherParentCreatorID uint32,
	otherParentIndex int,
	creatorID uint32) {
	e.Body.SelfParentIndex = selfParentIndex
	e.Body.OtherParentCreatorID = otherParentCreatorID
	e.Body.OtherParentIndex = otherParentIndex
	e.Body.CreatorID = creatorID
}

// WireBlockSignatures ...
func (e *Event) WireBlockSignatures() []WireBlockSignature {
	if e.Body.BlockSignatures != nil {
		wireSignatures := make([]WireBlockSignature, len(e.Body.BlockSignatures))

		for i, bs := range e.Body.BlockSignatures {
			wireSignatures[i] = bs.ToWire()
		}

		return wireSignatures
	}

	return nil
}

// ToWire ...
func (e *Event) ToWire() WireEvent {
	return WireEvent{
		Body: WireBody{
			Transactions:         e.Body.Transactions,
			InternalTransactions: e.Body.InternalTransactions,
			SelfParentIndex:      e.Body.SelfParentIndex,
			OtherParentCreatorID: e.Body.OtherParentCreatorID,
			OtherParentIndex:     e.Body.OtherParentIndex,
			CreatorID:            e.Body.CreatorID,
			Index:                e.Body.Index,
			BlockSignatures:      e.WireBlockSignatures(),
		},
		Signature: e.Signature,
	}
}

/*******************************************************************************
Sorting
*******************************************************************************/

// ByTopologicalOrder implements sort.Interface for []Event based on
// the TopologicalIndex field.
// THIS IS A PARTIAL ORDER
type ByTopologicalOrder []*Event

// Len ...
func (a ByTopologicalOrder) Len() int { return len(a) }

// Swap ...
func (a ByTopologicalOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less ...
func (a ByTopologicalOrder) Less(i, j int) bool {
	return a[i].TopologicalIndex < a[j].TopologicalIndex
}

// ByLamportTimestamp implements sort.Interface for []Event based on
// the lamportTimestamp field.
// THIS IS A TOTAL ORDER
type ByLamportTimestamp []*Event

// Len ...
func (a ByLamportTimestamp) Len() int { return len(a) }

// Swap ...
func (a ByLamportTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less ...
func (a ByLamportTimestamp) Less(i, j int) bool {
	it, jt := -1, -1
	if a[i].LamportTimestamp != nil {
		it = *a[i].LamportTimestamp
	}
	if a[j].LamportTimestamp != nil {
		jt = *a[j].LamportTimestamp
	}
	if it != jt {
		return it < jt
	}

	wsi, _ := hexutil.Decode(a[i].Signature)
	wsj, _ := hexutil.Decode(a[j].Signature)
	return new(big.Int).SetBytes(wsi).Cmp(new(big.Int).SetBytes(wsj)) < 0
}

// WireBody ...
type WireBody struct {
	Transactions         [][]byte
	InternalTransactions []InternalTransaction
	BlockSignatures      []WireBlockSignature

	CreatorID            uint32
	OtherParentCreatorID uint32
	Index                int
	SelfParentIndex      int
	OtherParentIndex     int
}

// WireEvent ...
type WireEvent struct {
	Body      WireBody
	Signature string
}

// BlockSignatures ...
func (we *WireEvent) BlockSignatures(validator []byte) []BlockSignature {
	if we.Body.BlockSignatures != nil {
		blockSignatures := make([]BlockSignature, len(we.Body.BlockSignatures))

		for k, bs := range we.Body.BlockSignatures {
			blockSignatures[k] = BlockSignature{
				Validator: validator,
				Index:     bs.Index,
				Signature: bs.Signature,
			}
		}

		return blockSignatures
	}

	return nil
}

//FrameEvent is a wrapper around a regular Event. It contains exported fields
//Round, Witness, and LamportTimestamp.
type FrameEvent struct {
	Core             *Event //EventBody + Signature
	Round            int
	LamportTimestamp int
	Witness          bool
}

//SortedFrameEvents implements sort.Interface for []FameEvent based on
//the lamportTimestamp field.
//THIS IS A TOTAL ORDER
type SortedFrameEvents []*FrameEvent

// Len ...
func (a SortedFrameEvents) Len() int { return len(a) }

// Swap ...
func (a SortedFrameEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less ...
func (a SortedFrameEvents) Less(i, j int) bool {
	if a[i].LamportTimestamp != a[j].LamportTimestamp {
		return a[i].LamportTimestamp < a[j].LamportTimestamp
	}

	wsi, _ := hexutil.Decode(a[i].Core.Signature)
	wsj, _ := hexutil.Decode(a[j].Core.Signature)
	return new(big.Int).SetBytes(wsi).Cmp(new(big.Int).SetBytes(wsj)) < 0
}
