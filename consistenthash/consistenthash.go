//go:build !solution

package consistenthash

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strconv"
)

const K = 256

type Node interface {
	// ID is some persistent and unique identifier
	ID() string
}

type ConsistentHash[N Node] struct {
	ring         []uint32            // sorted hashes
	hash2Node    map[uint32]*N       // hash -> node
	virtualNodes map[string][]uint32 // node ID -> hashes
	sorted       bool
}

func hashcode(key string) uint32 {
	// bad
	// var h uint32
	// for _, c := range key {
	// 	h = h*31 + uint32(c)
	// }
	// return h
	sum := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(sum[:4])
}

func New[N Node]() *ConsistentHash[N] {
	return &ConsistentHash[N]{
		ring:         []uint32{},
		hash2Node:    make(map[uint32]*N),
		virtualNodes: make(map[string][]uint32),
	}
}

func (h *ConsistentHash[N]) AddNode(n *N) {
	if _, exists := h.virtualNodes[(*n).ID()]; exists {
		return
	}

	// create K virtual nodes
	for i := range K {
		hash := hashcode((*n).ID() + ":" + strconv.Itoa(i))

		h.ring = append(h.ring, hash)
		h.hash2Node[hash] = n
		h.virtualNodes[(*n).ID()] = append(h.virtualNodes[(*n).ID()], hash)
	}

	h.sorted = false
}

func (h *ConsistentHash[N]) RemoveNode(n *N) {
	for _, hash := range h.virtualNodes[(*n).ID()] {
		delete(h.hash2Node, hash)
	}
	delete(h.virtualNodes, (*n).ID())

	// rebuild the ring
	h.ring = h.ring[:0]
	for hash := range h.hash2Node {
		h.ring = append(h.ring, hash)
	}

	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
	h.sorted = true
}

func (h *ConsistentHash[N]) GetNode(key string) *N {
	if len(h.ring) == 0 {
		return nil
	}

	if !h.sorted {
		sort.Slice(h.ring, func(i, j int) bool {
			return h.ring[i] < h.ring[j]
		})
		h.sorted = true
	}

	hash := hashcode(key)
	low := 0
	high := len(h.ring)
	for low < high {
		mid := (low + high) / 2
		if h.ring[mid] < hash {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low == len(h.ring) {
		low = 0
	}
	return h.hash2Node[h.ring[low]]
}
