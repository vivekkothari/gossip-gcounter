package main

import (
	"encoding/json"
	"github.com/hashicorp/memberlist"
	"gossip-gcounter/crdt"
)

type CRDTOp struct {
	Type     string `json:"type"` // "insert" or "delete"
	AfterID  string `json:"after_id,omitempty"`
	NewID    string `json:"new_id,omitempty"`
	Value    rune   `json:"value,omitempty"`
	TargetID string `json:"target_id,omitempty"` // for deletes
}

// CRDTEditorDelegate implements memberlist.Delegate interface
type CRDTEditorDelegate struct {
	editor     *crdt.CRDTText
	broadcasts *memberlist.TransmitLimitedQueue
}

func (d *CRDTEditorDelegate) NodeMeta(limit int) []byte { return nil }

func (d *CRDTEditorDelegate) NotifyMsg(msg []byte) {
	var op CRDTOp
	_ = json.Unmarshal(msg, &op)
	if op.Type == "insert" {
		d.editor.Insert(op.AfterID, op.NewID, op.Value)
	} else if op.Type == "delete" {
		d.editor.Delete(op.TargetID)
	}
}

func (d *CRDTEditorDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}
func (d *CRDTEditorDelegate) LocalState(join bool) []byte {
	//TODO implement me
	panic("implement me")
}

func (d *CRDTEditorDelegate) MergeRemoteState(buf []byte, join bool) {
	//TODO implement me
	panic("implement me")
}
