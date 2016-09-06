package vr

import (
	"context"
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

	recvc chan Msg // Transfer messages to state machine

	cancel context.CancelFunc
	stopc  chan struct{}
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
		recvc:  make(chan Msg),
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
			err := p.vr.Process(ctx, msg)
			if err != nil {
				log.Printf("failed to process messsage (%v)", err)
			}
		case <-p.stopc:
			return
		}
	}
}

func (p *peer) handleInbound() {
	for {
		msg := &Msg{}
		if err := p.dec.Decode(msg); err != nil {
			if err == io.EOF {
				p.stop()
				return
			}
			log.Printf("%s\n", err)
		}
		select {
		case p.recvc <- *msg:
			log.Printf("Replica(%d) received message from %d\n", msg.To, msg.From)
		case <-p.stopc:
			return
		}
	}
}

func (p *peer) send(m Msg) {
	log.Printf("Node (%d) sending to Node (%d)\n", m.From, m.To)
	if err := p.enc.Encode(m); err != nil {
		if err == io.EOF {
			p.stop()
		}
	}
	// log.Printf("***Node (%d) sent to Node (%d)***\n", m.From, m.To)
}

func (p *peer) stop() {
	close(p.stopc)
	_ = p.conn.Close()
}

func (p *peer) print() {
	fmt.Printf("Peer(%d, %s): %s\n", p.id, p.conn.LocalAddr().String(), p.conn.RemoteAddr().String())
}
