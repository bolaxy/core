package types

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/bolaxy/common"
	"github.com/bolaxy/conf"
	"github.com/bolaxy/errors"
)

// Key ...
type Key struct {
	X, Y string
}

// ToString ...
func (k Key) ToString() string {
	return fmt.Sprintf("{%s, %s}", k.X, k.Y)
}

// TreKey ...
type TreKey struct {
	X, Y, Z string
}

// ToString ...
func (k TreKey) ToString() string {
	return fmt.Sprintf("{%s, %s, %s}", k.X, k.Y, k.Z)
}

// ParticipantEventsCache ...
type ParticipantEventsCache struct {
	Participants *conf.PeerSet
	rim          *common.RollingIndexMap
}

// NewParticipantEventsCache ...
func NewParticipantEventsCache(size int) *ParticipantEventsCache {
	return &ParticipantEventsCache{
		Participants: conf.NewPeerSet([]*conf.Peer{}),
		rim:          common.NewRollingIndexMap("ParticipantEvents", size),
	}
}

// AddPeer ...
func (pec *ParticipantEventsCache) AddPeer(peer *conf.Peer) error {
	pec.Participants = pec.Participants.WithNewPeer(peer)
	return pec.rim.AddKey(peer.ID())
}

//particant is the CASE-INSENSITIVE string hex representation of the public key.
func (pec *ParticipantEventsCache) participantID(participant string) (uint32, error) {
	peer, ok := pec.Participants.ByPubKey[participant]
	if !ok {
		return 0, errors.NewStoreErr("ParticipantEvents", errors.UnknownParticipant, participant)
	}

	return peer.ID(), nil
}

//Get returns participant events with index > skip
func (pec *ParticipantEventsCache) Get(participant string, skipIndex int) ([]string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return []string{}, err
	}

	pe, err := pec.rim.Get(id, skipIndex)
	if err != nil {
		return []string{}, err
	}

	res := make([]string, len(pe))
	for k := 0; k < len(pe); k++ {
		res[k] = pe[k].(string)
	}
	return res, nil
}

// GetItem ...
func (pec *ParticipantEventsCache) GetItem(participant string, index int) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	item, err := pec.rim.GetItem(id, index)
	if err != nil {
		return "", err
	}
	return item.(string), nil
}

// GetLast ...
func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	last, err := pec.rim.GetLast(id)
	if err != nil {
		return "", err
	}

	return last.(string), nil
}

// Set ...
func (pec *ParticipantEventsCache) Set(participant string, hash string, index int) error {
	id, err := pec.participantID(participant)
	if err != nil {
		return err
	}

	return pec.rim.Set(id, hash, index)
}

// Known returns [participant id] => lastKnownIndex
func (pec *ParticipantEventsCache) Known() map[uint32]int {
	return pec.rim.Known()
}

// PeerSetCache ...
type PeerSetCache struct {
	rounds             sort.IntSlice
	peerSets           map[int]*conf.PeerSet
	repertoireByPubKey map[string]*conf.Peer
	repertoireByID     map[uint32]*conf.Peer
	firstRounds        map[uint32]int
}

// NewPeerSetCache ...
func NewPeerSetCache() *PeerSetCache {
	return &PeerSetCache{
		rounds:             sort.IntSlice{},
		peerSets:           make(map[int]*conf.PeerSet),
		repertoireByPubKey: make(map[string]*conf.Peer),
		repertoireByID:     make(map[uint32]*conf.Peer),
		firstRounds:        make(map[uint32]int),
	}
}

// Set ...
func (c *PeerSetCache) Set(round int, peerSet *conf.PeerSet) error {
	if _, ok := c.peerSets[round]; ok {
		return errors.NewStoreErr("PeerSetCache", errors.KeyAlreadyExists, strconv.Itoa(round))
	}

	c.peerSets[round] = peerSet

	c.rounds = append(c.rounds, round)
	c.rounds.Sort()

	for _, p := range peerSet.Peers {
		c.repertoireByPubKey[p.PubKeyString()] = p
		c.repertoireByID[p.ID()] = p
		fr, ok := c.firstRounds[p.ID()]
		if !ok || fr > round {
			c.firstRounds[p.ID()] = round
		}
	}

	return nil

}

// Get ...
func (c *PeerSetCache) Get(round int) (*conf.PeerSet, error) {
	//check if directly in peerSets
	ps, ok := c.peerSets[round]
	if ok {
		return ps, nil
	}

	//situate round in sorted rounds
	if len(c.rounds) == 0 {
		return nil, errors.NewStoreErr("PeerSetCache", errors.KeyNotFound, strconv.Itoa(round))
	}

	if round < c.rounds[0] {
		return c.peerSets[c.rounds[0]], nil
	}

	for i := 0; i < len(c.rounds)-1; i++ {
		if round >= c.rounds[i] && round < c.rounds[i+1] {
			return c.peerSets[c.rounds[i]], nil
		}
	}

	//return last PeerSet
	return c.peerSets[c.rounds[len(c.rounds)-1]], nil
}

// GetAll ...
func (c *PeerSetCache) GetAll() (map[int][]*conf.Peer, error) {
	res := make(map[int][]*conf.Peer)
	for _, r := range c.rounds {
		res[r] = c.peerSets[r].Peers
	}
	return res, nil
}

// RepertoireByID ...
func (c *PeerSetCache) RepertoireByID() map[uint32]*conf.Peer {
	return c.repertoireByID
}

// RepertoireByPubKey ...
func (c *PeerSetCache) RepertoireByPubKey() map[string]*conf.Peer {
	return c.repertoireByPubKey
}

// FirstRound ...
func (c *PeerSetCache) FirstRound(id uint32) (int, bool) {
	fr, ok := c.firstRounds[id]
	if ok {
		return fr, true
	}
	return math.MaxInt32, false
}

// PendingRound ...
type PendingRound struct {
	Index   int
	Decided bool
}

// OrderedPendingRounds ...
type OrderedPendingRounds []*PendingRound

// Len returns the length
func (a OrderedPendingRounds) Len() int { return len(a) }

// Swap swaps 2 elements
func (a OrderedPendingRounds) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less returns true if element i is less than element j.
func (a OrderedPendingRounds) Less(i, j int) bool {
	return a[i].Index < a[j].Index
}

// PendingRoundsCache ...
type PendingRoundsCache struct {
	items       map[int]*PendingRound
	sortedItems OrderedPendingRounds
}

// NewPendingRoundsCache ...
func NewPendingRoundsCache() *PendingRoundsCache {
	return &PendingRoundsCache{
		items:       make(map[int]*PendingRound),
		sortedItems: []*PendingRound{},
	}
}

// Queued ...
func (c *PendingRoundsCache) Queued(round int) bool {
	_, ok := c.items[round]
	return ok
}

// Set ...
func (c *PendingRoundsCache) Set(pendingRound *PendingRound) {
	c.items[pendingRound.Index] = pendingRound
	c.sortedItems = append(c.sortedItems, pendingRound)
	sort.Sort(c.sortedItems)
}

// GetOrderedPendingRounds ...
func (c *PendingRoundsCache) GetOrderedPendingRounds() OrderedPendingRounds {
	return c.sortedItems
}

// Update ...
func (c *PendingRoundsCache) Update(decidedRounds []int) {
	for _, drn := range decidedRounds {
		if dr, ok := c.items[drn]; ok {
			dr.Decided = true
		}
	}
}

// Clean ...
func (c *PendingRoundsCache) Clean(processedRounds []int) {
	for _, pr := range processedRounds {
		delete(c.items, pr)
	}
	newSortedItems := OrderedPendingRounds{}
	for _, pr := range c.items {
		newSortedItems = append(newSortedItems, pr)
	}
	sort.Sort(newSortedItems)
	c.sortedItems = newSortedItems
}

// SigPool ...
type SigPool struct {
	items map[string]BlockSignature
}

// NewSigPool ...
func NewSigPool() *SigPool {
	return &SigPool{
		items: make(map[string]BlockSignature),
	}
}

// Add ...
func (sp *SigPool) Add(blockSignature BlockSignature) {
	sp.items[blockSignature.Key()] = blockSignature
}

// Remove ...
func (sp *SigPool) Remove(key string) {
	delete(sp.items, key)
}

// RemoveSlice ...
func (sp *SigPool) RemoveSlice(sigs []BlockSignature) {
	for _, s := range sigs {
		delete(sp.items, s.Key())
	}
}

// Len ...
func (sp *SigPool) Len() int {
	return len(sp.items)
}

// Items ...
func (sp *SigPool) Items() map[string]BlockSignature {
	return sp.items
}

// Slice ...
func (sp *SigPool) Slice() []BlockSignature {
	res := []BlockSignature{}
	for _, bs := range sp.items {
		res = append(res, bs)
	}
	return res
}
