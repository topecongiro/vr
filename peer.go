package vr

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Peer gives an access to remote replicas.
type Peer interface {
	send(m Msg)
	stop()
	// for debug
	print()
}

// peer is a canonical implementation of Peer interface.
type peer struct {
	id ID
	vr Mediator

	status *peerStatus

	conn *net.TCPConn
	enc  *gob.Encoder
	dec  *gob.Decoder

	mu     sync.Mutex
	paused bool

	stopc chan struct{}
}

type peerStatus struct {
	id     ID
	mu     sync.Mutex
	active bool
	since  time.Time
}

func startPeer(tr *Transport, peerID ID, conn *net.TCPConn) *peer {
	status := &peerStatus{
		id: peerID,
	}
	p := &peer{
		id:     peerID,
		vr:     tr.VR,
		status: status,
		conn:   conn,
		enc:    gob.NewEncoder(conn),
		dec:    gob.NewDecoder(conn),
		stopc:  make(chan struct{}),
	}

	// On receiving message, transfer to the VR state machine.
	go p.handleInbound()

	return p
}

func (p *peer) handleInbound() {
	for {
		msg := Msg{}
		if err := p.dec.Decode(&msg); err != nil {
			if err == io.EOF {
				p.stop()
				return
			}
			p.print()
		}
		if err := p.vr.Process(msg); err != nil {
			log.Printf("failed to process messsage (%v)", err)
		}
	}
}

func (p *peer) send(m Msg) {
	if err := p.enc.Encode(m); err != nil {
		if err == io.EOF {
			p.stop()
		}
	}
}

func (p *peer) stop() {
	close(p.stopc)
	_ = p.conn.Close()
}

func (p *peer) print() {
	fmt.Printf("Peer(%d, %s): %s\n", p.id, p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
}
