package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

var (
	uploads      = make(map[uint64]*upload)
	uploadsMutex = new(sync.RWMutex)
)

type upload struct {
	id        uint64
	ufile     *file
	destroyed bool
	p         Progress
	to        <-chan time.Time
	mutex     *sync.RWMutex
}

func (u *upload) insert() {
	uploadsMutex.Lock()
	uploads[u.id] = u
	uploadsMutex.Unlock()
	u.destroyed = false
}

func (u *upload) delete() {
	uploadsMutex.Lock()
	delete(uploads, u.id)
	uploadsMutex.Unlock()
	u.destroyed = true
}

func (u *upload) timeout() {
	<-u.to
	u.mutex.Lock()
	if !u.p.Started {
		u.delete()
		log.Println("Timed Out")
	}
	u.mutex.Unlock()
}

type Progress struct {
	Uploaded  int64
	Total     int64
	Started   bool
	Completed bool
}

func makeUpload() uint64 {
	id := uint64(rand.Int31())
	for _, ok := uploads[id]; id == 0 && ok; _, ok = uploads[id] {
		id = uint64(rand.Int31())
	}
	u := new(upload)
	u.id = id
	u.mutex = new(sync.RWMutex)
	u.to = time.After(10 * time.Second)
	u.insert()
	go u.timeout()
	return id
}

func init() {
	rand.Seed(time.Now().Unix())
}
