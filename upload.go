package main

import (
	"math/rand"
	"sync"
)

var (
	uploads      = make(map[uint64]*upload)
	uploadsMutex = new(sync.RWMutex)
)

type upload struct {
	id        uint64
	ufile     file
	size      int64
	uploaded  int64
	completed bool
	mutex     *sync.RWMutex
}

func makeUpload() uint64 {
	id := uint64(rand.Int63())
	for _, ok := uploads[id]; id != 0 && !ok; _, ok = uploads[id] {
		id = uint64(rand.Int63())
	}
	mutex := new(sync.RWMutex)
	uploadsMutex.Lock()
	uploads[id] = &upload{id: id, mutex: mutex}
	uploadsMutex.Unlock()
	return id
}
