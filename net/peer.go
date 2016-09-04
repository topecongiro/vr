package net

import (
	"context"
	"encoding/gob"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/seuchi/vr"
)

// Peer gives an access to remote replicas.
type Peer interface {
	send(m vr.Msg)
	stop()
}

// peer is a canonical implementation of Peer interface.
type peer struct {
	id vr.ID
	vr vr.VR

	status *peerStatus

	conn *net.TCPConn

	mu     sync.Mutex
	paused bool

	recvc chan vr.Msg

	cancel context.CancelFunc
	stopc  chan struct{}
}

type peerStatus struct {
	id     vr.ID
	mu     sync.Mutex
	active bool
	since  time.Time
}

func startPeer(tr *Transport, peerID vr.ID, conn *net.TCPConn) *peer {
	status := &peerStatus{
		id: peerID,
	}
	p := &peer{
		id:     peerID,
		vr:     tr.VR,
		status: status,
		conn:   conn,
		recvc:  make(chan vr.Msg),
		stopc:  make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	// On receiving message, transfer to the VR state machine.
	go p.transferToVR(ctx)
	go p.handleInbound()

	return p
}

func (p *peer) transferToVR(ctx context.Context) {
	for {
		select {
		case msg := <-p.recvc:
			if err := p.vr.Process(ctx, msg); err != nil {
				log.Printf("failed to process messsage (%v)", err)
			}
		case <-p.stopc:
			return
		}
	}
}

func (p *peer) handleInbound() {
	dec := gob.NewDecoder(p.conn)
	for {
		var msg vr.Msg
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				p.stop()
				return
			}
		}
		select {
		case p.recvc <- msg:
		case <-p.stopc:
			return
		}
	}
}

func (p *peer) stop() {
	close(p.stopc)
	_ = p.conn.Close()
}

func (p *peer) send(m vr.Msg) {
	enc := gob.NewEncoder(p.conn)
	if err := enc.Encode(m); err != nil {
		if err == io.EOF {
			p.stop()
		}
	}
}
