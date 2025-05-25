package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/hashicorp/memberlist"
	"gossip-gcounter/crdt"
	"log"
	"os"
	"strconv"
	"time"
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
	gob.NewEncoder(&buf).Encode(d.counter.Snapshot())
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

func main() {
	// Get container hostname as nodeID
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("Could not get hostname:", err)
	}

	// Number of replicas - read from env or default 3
	numReplicas := 0
	if v := os.Getenv("TOTAL_NODES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			numReplicas = n
		}
	}

	fmt.Println("Num replicas is ", numReplicas)

	// Gossip port inside container and Docker network
	const basePort = 7946
	bindPort := basePort // all containers bind to same port inside Docker network

	// Prepare peers list excluding self
	peerService := os.Getenv("PEER_SERVICE")
	if peerService == "" {
		peerService = "node" // default
	}
	// Build peer list excluding self
	var peers []string
	for i := 1; i <= numReplicas; i++ {
		peerName := fmt.Sprintf("gossip-gcounter-%s-%d:%d", peerService, i, basePort)
		peers = append(peers, peerName)
	}
	fmt.Println("Peers to join:", peers)

	// Setup CRDT and memberlist delegate
	gob.Register(map[string]int{})
	counter := crdt.NewGCounter(hostname)
	delegate := &CRDTDelegate{counter: counter}
	delegate.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return numReplicas },
		RetransmitMult: 3,
	}

	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = hostname
	cfg.BindPort = bindPort
	cfg.BindAddr = "0.0.0.0"
	cfg.Delegate = delegate

	list, err := memberlist.Create(cfg)
	if err != nil {
		log.Fatal("Failed to create memberlist:", err)
	}

	if len(peers) > 0 {
		n, err := list.Join(peers)
		if err != nil {
			log.Println("Join failed:", err)
		} else {
			log.Printf("Joined %d peers\n", n)
		}
	}

	// Periodic increment + broadcast
	go func() {
		for {
			time.Sleep(3 * time.Second)
			counter.Increment()

			var buf bytes.Buffer
			gob.NewEncoder(&buf).Encode(counter.Snapshot())
			msg := &crdt.Broadcast{Msg: buf.Bytes()}
			delegate.broadcasts.QueueBroadcast(msg)

			fmt.Printf("[%s] Counter Value: %d\n", hostname, counter.Value())
		}
	}()

	select {}
}
