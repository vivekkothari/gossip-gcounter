package crdt

import (
	"github.com/hashicorp/memberlist"
)

type Broadcast struct {
	Msg    []byte
	notify chan struct{}
	msg    []byte
}

func (b *Broadcast) Invalidates(other memberlist.Broadcast) bool { return false }
func (b *Broadcast) Message() []byte                             { return b.Msg }
func (b *Broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}
