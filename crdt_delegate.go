package main

import (
	"bytes"
	"encoding/gob"
	"github.com/hashicorp/memberlist"
	"gossip-gcounter/crdt"
	"log"
)

// CRDTDelegate implements memberlist.Delegate interface
type CRDTDelegate struct {
	counter    *crdt.GCounter
	broadcasts *memberlist.TransmitLimitedQueue
}

func (d *CRDTDelegate) NodeMeta(limit int) []byte { return nil }
func (d *CRDTDelegate) NotifyMsg(b []byte) {
	var incoming map[string]int
	buf := bytes.NewBuffer(b)
	if err := gob.NewDecoder(buf).Decode(&incoming); err != nil {
		log.Println("decode error:", err)
		return
	}
	d.counter.Merge(incoming)
}
func (d *CRDTDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}
func (d *CRDTDelegate) LocalState(join bool) []byte {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(d.counter.Snapshot())
	if err != nil {
		return nil
	}
	return buf.Bytes()
}
func (d *CRDTDelegate) MergeRemoteState(buf []byte, join bool) {
	var incoming map[string]int
	if err := gob.NewDecoder(bytes.NewReader(buf)).Decode(&incoming); err != nil {
		log.Println("merge decode error:", err)
		return
	}
	d.counter.Merge(incoming)
}
