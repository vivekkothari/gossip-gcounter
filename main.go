package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"gossip-gcounter/crdt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var doc = &crdt.CRDTText{}

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
	editorDelegate := &CRDTEditorDelegate{editor: doc}
	delegate.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return numReplicas },
		RetransmitMult: 3,
	}
	editorDelegate.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return numReplicas },
		RetransmitMult: 3,
	}

	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = hostname
	cfg.BindPort = basePort
	cfg.BindAddr = "0.0.0.0"
	cfg.Delegate = editorDelegate

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

	// Start REST API
	http.HandleFunc("/increment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		counter.Increment()

		// Optionally broadcast the delta
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(counter.Delta()); err == nil {
			msg := &crdt.Broadcast{Msg: buf.Bytes()}
			delegate.broadcasts.QueueBroadcast(msg)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Incremented")
	})

	http.HandleFunc("/counters", func(w http.ResponseWriter, r *http.Request) {
		state := counter.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	})

	http.HandleFunc("/value", func(w http.ResponseWriter, r *http.Request) {
		value := counter.Value()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"value": value})
	})

	http.HandleFunc("/insert", insertHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/text", getTextHandler)

	// Run HTTP server on port 9002
	go func() {
		log.Println("Starting REST API on :9002")
		if err := http.ListenAndServe(":9002", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Periodic increment + broadcast
	go func() {
		for {
			time.Sleep(3 * time.Second)
			//counter.Increment()

			var buf bytes.Buffer
			err := gob.NewEncoder(&buf).Encode(counter.Delta())
			if err != nil {
				log.Println("delta encode error:", err)
				continue
			}
			msg := &crdt.Broadcast{Msg: buf.Bytes()}
			delegate.broadcasts.QueueBroadcast(msg)

			fmt.Printf("[%s] Counter Value: %d\n", hostname, counter.Value())
		}
	}()

	select {}
}

func insertHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AfterID string `json:"after_id"`
		NewID   string `json:"new_id"`
		Value   string `json:"value"` // single character
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Value) != 1 {
		http.Error(w, "value must be a single character", http.StatusBadRequest)
		return
	}

	doc.Insert(req.AfterID, req.NewID, rune(req.Value[0]))
	w.WriteHeader(http.StatusOK)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	doc.Delete(req.ID)
	w.WriteHeader(http.StatusOK)
}

func getTextHandler(w http.ResponseWriter, r *http.Request) {
	text := doc.GetVisibleText()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": text})
}
